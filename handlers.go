package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
)

var (
	tmplCatalog  = template.Must(template.ParseFiles("templates/catalog.html"))
	tmplCart     = template.Must(template.ParseFiles("templates/cart.html"))
	tmplOrder    = template.Must(template.ParseFiles("templates/order.html"))
	tmplAccount  = template.Must(template.ParseFiles("templates/account.html"))
	tmplRegister = template.Must(template.ParseFiles("templates/register.html"))
	tmplLogin    = template.Must(template.ParseFiles("templates/login.html"))
)

// ---------- Каталог ----------

type CatalogData struct {
	CurrentUser *User
	Products    []Product
}

func catalogHandler(w http.ResponseWriter, r *http.Request) {
	products, err := listProducts()
	if err != nil {
		log.Println(err)
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	tmplCatalog.Execute(w, CatalogData{
		CurrentUser: currentUser(r),
		Products:    products,
	})
}

// ---------- Корзина ----------

func cartAddHandler(w http.ResponseWriter, r *http.Request, user *User) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if _, found := findProductByID(id); !found {
		http.NotFound(w, r)
		return
	}

	if err := addToCart(user.ID, id); err != nil {
		log.Println(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type CartData struct {
	CurrentUser *User
	Lines       []CartLine
	Total       int
	Error       string
}

func cartHandler(w http.ResponseWriter, r *http.Request, user *User) {
	lines, total, err := getCart(user.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	errMsg := ""
	if r.URL.Query().Get("error") == "stock" {
		errMsg = "Недостаточно товара на складе — уменьшите количество"
	}

	tmplCart.Execute(w, CartData{
		CurrentUser: user,
		Lines:       lines,
		Total:       total,
		Error:       errMsg,
	})
}

func cartRemoveHandler(w http.ResponseWriter, r *http.Request, user *User) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := removeFromCart(user.ID, id); err != nil {
		log.Println(err)
	}

	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

// cartUpdateHandler — меняет количество конкретного товара в корзине
// на значение, введённое пользователем в поле формы.
func cartUpdateHandler(w http.ResponseWriter, r *http.Request, user *User) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	quantity, err := strconv.Atoi(r.FormValue("quantity"))
	if err != nil {
		// Некорректный ввод — просто возвращаемся в корзину без изменений
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	if err := updateCartQuantity(user.ID, id, quantity); err != nil {
		log.Println(err)
	}

	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

// ---------- Оформление заказа ----------

func checkoutHandler(w http.ResponseWriter, r *http.Request, user *User) {
	orderID, err := checkout(user.ID)
	if err != nil {
		log.Println("оформление заказа не удалось:", err)
		if err == errOutOfStock {
			http.Redirect(w, r, "/cart?error=stock", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/cart", http.StatusSeeOther)
		}
		return
	}

	http.Redirect(w, r, "/order/"+strconv.Itoa(orderID), http.StatusSeeOther)
}

type OrderData struct {
	CurrentUser *User
	Order       Order
	Lines       []OrderLine
}

func orderHandler(w http.ResponseWriter, r *http.Request, user *User) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	order, found := getOrder(id)
	if !found {
		http.NotFound(w, r)
		return
	}

	// Проверка доступа: чужой заказ смотреть нельзя
	if order.UserID != user.ID {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	lines, err := getOrderLines(id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	tmplOrder.Execute(w, OrderData{
		CurrentUser: user,
		Order:       order,
		Lines:       lines,
	})
}

// ---------- Личный кабинет ----------

type AccountData struct {
	CurrentUser *User
	Orders      []Order
}

func accountHandler(w http.ResponseWriter, r *http.Request, user *User) {
	orders, err := listOrdersByUser(user.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	tmplAccount.Execute(w, AccountData{
		CurrentUser: user,
		Orders:      orders,
	})
}

// ---------- Регистрация / вход ----------

type AuthData struct {
	CurrentUser *User
	Error       string
}

func registerFormHandler(w http.ResponseWriter, r *http.Request) {
	tmplRegister.Execute(w, AuthData{})
}

func registerSubmitHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		tmplRegister.Execute(w, AuthData{Error: "Заполните все поля"})
		return
	}

	if _, exists := findUserByUsername(username); exists {
		tmplRegister.Execute(w, AuthData{Error: "Такой пользователь уже существует"})
		return
	}

	salt, err := generateSalt()
	if err != nil {
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	newUser, err := createUser(username, hashPassword(password, salt), salt)
	if err != nil {
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if err := createSession(w, newUser.ID); err != nil {
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func loginFormHandler(w http.ResponseWriter, r *http.Request) {
	tmplLogin.Execute(w, AuthData{})
}

func loginSubmitHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, found := findUserByUsername(username)
	if !found || hashPassword(password, user.Salt) != user.PasswordHash {
		tmplLogin.Execute(w, AuthData{Error: "Неверный логин или пароль"})
		return
	}

	if err := createSession(w, user.ID); err != nil {
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	clearSession(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
