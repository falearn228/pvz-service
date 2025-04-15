package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// PasswordCheckerInterface определяет интерфейс для проверки паролей
type PasswordCheckerInterface interface {
	CheckPassword(password, hashedPassword string) error
}

// DefaultPasswordChecker реализует стандартную проверку с bcrypt
// создание пустой структуры и интерфейса нужно, чтобы легче реализовать мокирование
// в тестах
type DefaultPasswordChecker struct{}

func (c *DefaultPasswordChecker) CheckPassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// HashPassword создает хеш пароля с использованием bcrypt
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}
