package utils

import (
	"fmt"
	"time"

	"pvz-service/internal/config"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

// JWTManager управляет созданием и проверкой JWT токенов
type JWTManager struct {
	secretKey  string
	expireTime time.Duration
}

// NewJWTManager создает новый экземпляр JWTManager
func NewJWTManager(config *config.JWTConfig) *JWTManager {
	return &JWTManager{
		secretKey:  config.Secret,
		expireTime: config.ExpireTime,
	}
}

// CustomClaims представляет данные, которые будут закодированы в JWT
type CustomClaims struct {
	jwt.StandardClaims
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// GenerateDummyToken создает тестовый JWT токен для указанной роли
func (manager *JWTManager) GenerateDummyToken(role string) (string, error) {
	// Создаем уникальный ID для пользователя
	dummyUserID := uuid.New().String()

	// Устанавливаем время истечения токена
	expirationTime := time.Now().Add(manager.expireTime)

	// Создаем claims
	claims := &CustomClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Subject:   dummyUserID,
		},
		UserID: dummyUserID,
		Role:   role,
	}

	// Создаем токен с claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен нашим секретным ключом
	tokenString, err := token.SignedString([]byte(manager.secretKey))

	return tokenString, err
}

// ValidateToken проверяет JWT токен
func (manager *JWTManager) ValidateToken(tokenString string) (*CustomClaims, error) {
	// Парсим токен
	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(manager.secretKey), nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Проверяем claims
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
