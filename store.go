package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

// ---------- Модели ----------

type User struct {
	ID           int
	Username     string
	PasswordHash string
	Salt         string
}

type Product struct {
	ID          int
	Title       string
	Description string
	Price       int
	ImagePath   string
	Stock       int
}

// CartLine — товар в корзине вместе с данными из products (результат JOIN)
type CartLine struct {
	ProductID int
	Title     string
	Price     int
	ImagePath string
	Quantity  int
	Subtotal  int
}

type Order struct {
	ID        int
	UserID    int
	CreatedAt string
	Total     int
}

// OrderLine — "снимок" товара на момент покупки (см. пояснение при разборе)
type OrderLine struct {
	Title    string
	Price    int
	Quantity int
	Subtotal int
}

var db *sql.DB

var (
	errEmptyCart  = errors.New("корзина пуста")
	errOutOfStock = errors.New("товара не осталось на складе")
)

func initDB() {
	// DB_PATH позволяет указать, где хранить файл БД — например, на Railway это
	// будет путь к смонтированному постоянному диску (Volume). Локально переменной
	// нет, и файл, как раньше, создаётся прямо в папке проекта.
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "shop.db"
	}

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		salt TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT,
		price INTEGER NOT NULL,
		image_path TEXT NOT NULL,
		stock INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS cart_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id),
		product_id INTEGER NOT NULL REFERENCES products(id),
		quantity INTEGER NOT NULL DEFAULT 1,
		UNIQUE(user_id, product_id)
	);

	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id),
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		total INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS order_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id INTEGER NOT NULL REFERENCES orders(id),
		product_id INTEGER,
		title TEXT NOT NULL,
		price INTEGER NOT NULL,
		quantity INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		expires_at INTEGER NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		log.Fatal(err)
	}
}

// seedProducts — наполняет каталог парой товаров при первом запуске
func seedProducts() {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&count); err != nil {
		log.Fatal(err)
	}
	if count > 0 {
		return
	}

	demo := []Product{
		{Title: "Механическая клавиатура", Description: "Компактная клавиатура с тактильными переключателями", Price: 6500, ImagePath: "https://loremflickr.com/400/300/mechanical,keyboard", Stock: 12},
		{Title: "Беспроводные наушники", Description: "Активное шумоподавление, до 30 часов работы", Price: 9200, ImagePath: "https://loremflickr.com/400/300/wireless,headphones", Stock: 8},
		{Title: "Настольная лампа", Description: "Регулируемая яркость, USB-зарядка", Price: 2300, ImagePath: "https://loremflickr.com/400/300/desk,lamp", Stock: 20},
		{Title: "Рюкзак для ноутбука", Description: "Влагостойкий, отделение под 15-дюймовый ноутбук", Price: 3400, ImagePath: "https://loremflickr.com/400/300/laptop,backpack", Stock: 15},
		{Title: "Термокружка", Description: "Держит тепло 6 часов, объём 450мл", Price: 1200, ImagePath: "https://loremflickr.com/400/300/thermos,mug", Stock: 30},
		{Title: "Настольный органайзер", Description: "Дерево, 5 отделений", Price: 1800, ImagePath: "https://loremflickr.com/400/300/desk,organizer", Stock: 0},
	}

	for _, p := range demo {
		_, err := db.Exec(
			`INSERT INTO products (title, description, price, image_path, stock) VALUES (?, ?, ?, ?, ?)`,
			p.Title, p.Description, p.Price, p.ImagePath, p.Stock,
		)
		if err != nil {
			log.Println("не удалось добавить товар:", err)
		}
	}
}

func generateSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashPassword(password, salt string) string {
	h := sha256.Sum256([]byte(password + salt))
	return hex.EncodeToString(h[:])
}

// ---------- Пользователи ----------

func findUserByUsername(username string) (User, bool) {
	var u User
	err := db.QueryRow(
		`SELECT id, username, password_hash, salt FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Salt)
	if err != nil {
		return User{}, false
	}
	return u, true
}

func findUserByID(id int) (User, bool) {
	var u User
	err := db.QueryRow(
		`SELECT id, username, password_hash, salt FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Salt)
	if err != nil {
		return User{}, false
	}
	return u, true
}

func createUser(username, passwordHash, salt string) (User, error) {
	res, err := db.Exec(
		`INSERT INTO users (username, password_hash, salt) VALUES (?, ?, ?)`,
		username, passwordHash, salt,
	)
	if err != nil {
		return User{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return User{ID: int(id), Username: username, PasswordHash: passwordHash, Salt: salt}, nil
}

// ---------- Каталог ----------

func listProducts() ([]Product, error) {
	rows, err := db.Query(`SELECT id, title, description, price, image_path, stock FROM products ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Price, &p.ImagePath, &p.Stock); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func findProductByID(id int) (Product, bool) {
	var p Product
	err := db.QueryRow(
		`SELECT id, title, description, price, image_path, stock FROM products WHERE id = ?`,
		id,
	).Scan(&p.ID, &p.Title, &p.Description, &p.Price, &p.ImagePath, &p.Stock)
	if err != nil {
		return Product{}, false
	}
	return p, true
}

// ---------- Корзина ----------

// addToCart — если товара ещё нет в корзине, добавляет со количеством 1,
// если уже есть — увеличивает количество на 1 (это и есть "upsert")
func addToCart(userID, productID int) error {
	_, err := db.Exec(`
		INSERT INTO cart_items (user_id, product_id, quantity)
		VALUES (?, ?, 1)
		ON CONFLICT(user_id, product_id) DO UPDATE SET quantity = quantity + 1
	`, userID, productID)
	return err
}

func removeFromCart(userID, productID int) error {
	_, err := db.Exec(`DELETE FROM cart_items WHERE user_id = ? AND product_id = ?`, userID, productID)
	return err
}

func getCart(userID int) ([]CartLine, int, error) {
	rows, err := db.Query(`
		SELECT p.id, p.title, p.price, p.image_path, ci.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.user_id = ?
		ORDER BY ci.id
	`, userID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	lines := make([]CartLine, 0)
	total := 0
	for rows.Next() {
		var l CartLine
		if err := rows.Scan(&l.ProductID, &l.Title, &l.Price, &l.ImagePath, &l.Quantity); err != nil {
			return nil, 0, err
		}
		l.Subtotal = l.Price * l.Quantity
		total += l.Subtotal
		lines = append(lines, l)
	}
	return lines, total, rows.Err()
}

// ---------- Оформление заказа ----------

// checkout — переносит содержимое корзины в новый заказ ОДНОЙ транзакцией:
// проверяет остатки на складе, создаёт заказ и его позиции, списывает склад,
// очищает корзину. Если что-то не так — откатывается всё целиком.
func checkout(userID int) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT p.id, p.title, p.price, p.stock, ci.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.user_id = ?
	`, userID)
	if err != nil {
		return 0, err
	}

	type line struct {
		productID, price, stock, quantity int
		title                             string
	}
	var lines []line
	total := 0

	for rows.Next() {
		var l line
		if err := rows.Scan(&l.productID, &l.title, &l.price, &l.stock, &l.quantity); err != nil {
			rows.Close()
			return 0, err
		}
		lines = append(lines, l)
		total += l.price * l.quantity
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(lines) == 0 {
		return 0, errEmptyCart
	}

	for _, l := range lines {
		if l.stock < l.quantity {
			return 0, errOutOfStock
		}
	}

	res, err := tx.Exec(`INSERT INTO orders (user_id, total) VALUES (?, ?)`, userID, total)
	if err != nil {
		return 0, err
	}
	orderID64, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	orderID := int(orderID64)

	for _, l := range lines {
		if _, err := tx.Exec(
			`INSERT INTO order_items (order_id, product_id, title, price, quantity) VALUES (?, ?, ?, ?, ?)`,
			orderID, l.productID, l.title, l.price, l.quantity,
		); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(`UPDATE products SET stock = stock - ? WHERE id = ?`, l.quantity, l.productID); err != nil {
			return 0, err
		}
	}

	if _, err := tx.Exec(`DELETE FROM cart_items WHERE user_id = ?`, userID); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return orderID, nil
}

func getOrder(orderID int) (Order, bool) {
	var o Order
	err := db.QueryRow(
		`SELECT id, user_id, created_at, total FROM orders WHERE id = ?`,
		orderID,
	).Scan(&o.ID, &o.UserID, &o.CreatedAt, &o.Total)
	if err != nil {
		return Order{}, false
	}
	return o, true
}

func getOrderLines(orderID int) ([]OrderLine, error) {
	rows, err := db.Query(
		`SELECT title, price, quantity FROM order_items WHERE order_id = ?`,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]OrderLine, 0)
	for rows.Next() {
		var l OrderLine
		if err := rows.Scan(&l.Title, &l.Price, &l.Quantity); err != nil {
			return nil, err
		}
		l.Subtotal = l.Price * l.Quantity
		lines = append(lines, l)
	}
	return lines, rows.Err()
}
