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

	// Каталог
	mux.HandleFunc("GET /", catalogHandler)

	// Корзина
	mux.HandleFunc("POST /cart/add/{id}", cartAddHandler)
	mux.HandleFunc("GET /cart", cartHandler)
	mux.HandleFunc("POST /cart/remove/{id}", cartRemoveHandler)
	mux.HandleFunc("POST /cart/checkout", checkoutHandler)

	// Заказ
	mux.HandleFunc("GET /order/{id}", orderHandler)

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
