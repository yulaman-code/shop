package main

import (
	"time"
)

type Review struct {
	ID        int
	Rating    int
	Stars     string
	Comment   string
	Username  string
	CreatedAt time.Time
}

func starsString(rating int) string {
	filled := ""
	for i := 0; i < rating; i++ {
		filled += "★"
	}
	for i := rating; i < 5; i++ {
		filled += "☆"
	}
	return filled
}

func hasPurchased(userID, productID int) bool {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*)
		 FROM order_items
		 JOIN orders ON order_items.order_id = orders.id
		 WHERE orders.user_id = ? AND order_items.product_id = ?`,
		userID, productID,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func hasReviewed(userID, productID int) bool {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM reviews WHERE user_id = ? AND product_id = ?`,
		userID, productID,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func addReview(productID, userID, rating int, comment string) error {
	_, err := db.Exec(
		`INSERT INTO reviews (product_id, user_id, rating, comment, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		productID, userID, rating, comment, time.Now().Unix(),
	)
	return err
}

func listReviews(productID int) ([]Review, error) {
	rows, err := db.Query(
		`SELECT reviews.id, reviews.rating, reviews.comment, users.username, reviews.created_at
		 FROM reviews
		 JOIN users ON reviews.user_id = users.id
		 WHERE reviews.product_id = ?
		 ORDER BY reviews.created_at DESC`,
		productID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := make([]Review, 0)
	for rows.Next() {
		var r Review
		var ts int64
		if err := rows.Scan(&r.ID, &r.Rating, &r.Comment, &r.Username, &ts); err != nil {
			return nil, err
		}
		r.Stars = starsString(r.Rating)
		r.CreatedAt = time.Unix(ts, 0)
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}

func averageRating(productID int) (float64, int) {
	var avg float64
	var count int
	err := db.QueryRow(
		`SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM reviews WHERE product_id = ?`,
		productID,
	).Scan(&avg, &count)
	if err != nil {
		return 0, 0
	}
	return avg, count
}
