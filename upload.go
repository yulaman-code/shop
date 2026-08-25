package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func uploadDir() string {
	dir := os.Getenv("UPLOAD_DIR")
	if dir == "" {
		dir = "static/images"
	}
	return dir
}

func saveUploadedImage(r *http.Request, formField string) (string, error) {
	file, header, err := r.FormFile(formField)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		return "", errors.New("недопустимый формат файла")
	}

	dir := uploadDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	name := strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	return "/uploads/" + name, nil
}
