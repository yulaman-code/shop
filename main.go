package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	initDB()
	defer db.Close()
	seedProducts()

	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir()))))

	// Каталог
	mux.HandleFunc("GET /", catalogHandler)
	mux.HandleFunc("GET /product/{id}", productHandler)
	mux.HandleFunc("POST /product/{id}/review", requireAuth(reviewSubmitHandler))

	// Корзина (доступна только залогиненным — requireAuth проверяет это один раз)
	mux.HandleFunc("POST /cart/add/{id}", requireAuth(cartAddHandler))
	mux.HandleFunc("GET /cart", requireAuth(cartHandler))
	mux.HandleFunc("POST /cart/remove/{id}", requireAuth(cartRemoveHandler))
	mux.HandleFunc("POST /cart/update/{id}", requireAuth(cartUpdateHandler))
	mux.HandleFunc("POST /cart/checkout", requireAuth(checkoutHandler))

	// Заказ (тоже только для залогиненных)
	mux.HandleFunc("GET /order/{id}", requireAuth(orderHandler))

	// Личный кабинет — история заказов
	mux.HandleFunc("GET /account", requireAuth(accountHandler))

	mux.HandleFunc("GET /admin", requireAdmin(adminHandler))
	mux.HandleFunc("POST /admin/add", requireAdmin(adminAddHandler))
	mux.HandleFunc("POST /admin/save-all", requireAdmin(adminSaveAllHandler))
	mux.HandleFunc("POST /admin/image/{id}", requireAdmin(adminImageHandler))

	// Пользователи
	mux.HandleFunc("GET /register", registerFormHandler)
	mux.HandleFunc("POST /register", registerSubmitHandler)
	mux.HandleFunc("GET /login", loginFormHandler)
	mux.HandleFunc("POST /login", loginSubmitHandler)
	mux.HandleFunc("GET /logout", logoutHandler)

	// Railway (и большинство хостингов) сообщают порт через переменную окружения PORT.
	// Локально её нет — тогда используем 8080, как и раньше.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Сервер запущен на порту " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
