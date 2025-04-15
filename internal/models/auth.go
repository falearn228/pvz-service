package models

// Типы пользователей
const (
	RoleEmployee  = "employee"
	RoleModerator = "moderator"
)

// User представляет пользователя в системе
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	PasswordHash string `json:"-" db:"password_hash"` // Не отдаем пароль в JSON
}

// DummyLoginRequest представляет запрос на получение временного токена
type DummyLoginRequest struct {
	Role string `json:"role" binding:"required,oneof=employee moderator"`
}

// DummyLoginResponse представляет ответ с токеном авторизации
type DummyLoginResponse struct {
	Token string `json:"token"`
}

// ErrorResponse представляет ошибку API
type ErrorResponse struct {
	Message string `json:"message"`
}

// internal/models/models.go

// RegisterRequest представляет запрос на регистрацию пользователя
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=employee moderator"`
}

// RegisterResponse представляет ответ на запрос регистрации
type RegisterResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// LoginRequest представляет запрос на авторизацию
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse представляет ответ с токеном авторизации
type LoginResponse struct {
	Token string `json:"token"`
}
