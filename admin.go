package main

func createProduct(title, description string, price int, imagePath string, stock int) error {
	_, err := db.Exec(
		`INSERT INTO products (title, description, price, image_path, stock)
		 VALUES (?, ?, ?, ?, ?)`,
		title, description, price, imagePath, stock,
	)
	return err
}

func updateProduct(id, price, stock int) error {
	_, err := db.Exec(
		`UPDATE products SET price = ?, stock = ? WHERE id = ?`,
		price, stock, id,
	)
	return err
}

func updateProductImage(id int, imagePath string) error {
	_, err := db.Exec(
		`UPDATE products SET image_path = ? WHERE id = ?`,
		imagePath, id,
	)
	return err
}
