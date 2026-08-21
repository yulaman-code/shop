package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// createSession — создаёт запись в таблице sessions (а не в памяти процесса),
// поэтому сессия переживает перезапуск сервера.
func createSession(w http.ResponseWriter, userID int) error {
	token, err := generateToken()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	_, err = db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt.Unix(),
	)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Expires:  expiresAt,
	})
	return nil
}

func clearSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		db.Exec(`DELETE FROM sessions WHERE token = ?`, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})
}

// currentUser — теперь смотрит не в карту в памяти, а в таблицу sessions.
// Дополнительно проверяет, не истёк ли срок действия токена.
func currentUser(r *http.Request) *User {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil
	}

	var userID int
	var expiresAt int64

	err = db.QueryRow(
		`SELECT user_id, expires_at FROM sessions WHERE token = ?`,
		cookie.Value,
	).Scan(&userID, &expiresAt)
	if err != nil {
		return nil
	}

	if time.Now().Unix() > expiresAt {
		// Токен просрочен — удаляем его, чтобы таблица не копила мусор
		db.Exec(`DELETE FROM sessions WHERE token = ?`, cookie.Value)
		return nil
	}

	u, found := findUserByID(userID)
	if !found {
		return nil
	}
	return &u
}

// requireAuth — middleware. Оборачивает обработчик, которому обязательно
// нужен залогиненный пользователь. Сама проверка "залогинен ли" происходит
// один раз, здесь, а не повторяется внутри каждого такого обработчика.
//
// authedHandler — это обработчик с ДОПОЛНИТЕЛЬНЫМ параметром *User: раз
// requireAuth уже нашёл пользователя, он сразу передаёт его дальше — самому
// обработчику не нужно ещё раз спрашивать currentUser(r).
type authedHandler func(w http.ResponseWriter, r *http.Request, user *User)

func requireAuth(next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r, user)
	}
}
