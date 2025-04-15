package middleware

import (
	"net/http"
	"pvz-service/internal/models"
	"pvz-service/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware создает middleware для проверки JWT токена
func AuthMiddleware(jwtManager utils.JWTManagerInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем токен из заголовка Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Message: "Отсутствует токен авторизации",
			})
			c.Abort()
			return
		}

		// Извлекаем токен из заголовка
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Message: "Неверный формат токена",
			})
			c.Abort()
			return
		}
		tokenString := tokenParts[1]

		// Проверяем токен
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Message: "Неверный токен: " + err.Error(),
			})
			c.Abort()
			return
		}

		// Сохраняем данные пользователя в контексте
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)

		c.Next()
	}
}

// RequireRole создает middleware для проверки роли пользователя
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем роль пользователя из контекста
		userRole, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Message: "Нет данных о пользователе",
			})
			c.Abort()
			return
		}

		// Проверяем соответствие роли
		if userRole != requiredRole {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Message: "Доступ запрещен: недостаточно прав",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
