package handlers

import (
	"net/http"

	"pvz-service/internal/db/queries"
	"pvz-service/internal/models"
	"pvz-service/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthHandler содержит обработчики для авторизации
type AuthHandler struct {
	jwtManager  *utils.JWTManager
	authQueries *queries.AuthQueries
}

// NewAuthHandler создает новый экземпляр AuthHandler
func NewAuthHandler(jwtManager *utils.JWTManager, authQueries *queries.AuthQueries) *AuthHandler {
	return &AuthHandler{
		jwtManager:  jwtManager,
		authQueries: authQueries,
	}
}

// DummyLogin обрабатывает запрос на получение тестового токена
func (h *AuthHandler) DummyLogin(c *gin.Context) {
	var req models.LoginRequest

	// Проверяем запрос
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Неверный запрос: " + err.Error(),
		})
		return
	}

	// Генерируем JWT токен
	token, err := h.jwtManager.GenerateDummyToken(req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка генерации токена: " + err.Error(),
		})
		return
	}

	// Возвращаем токен
	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
	})
}

// Register обрабатывает запрос на регистрацию пользователя
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest

	// Проверяем данные запроса
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Неверный запрос: " + err.Error(),
		})
		return
	}

	// Проверяем, существует ли пользователь с таким email
	exists, err := h.authQueries.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при проверке email: " + err.Error(),
		})
		return
	}

	if exists {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Пользователь с таким email уже существует",
		})
		return
	}

	// Хешируем пароль
	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при хешировании пароля: " + err.Error(),
		})
		return
	}

	// Создаем пользователя
	id, err := h.authQueries.CreateUser(c.Request.Context(), req.Email, passwordHash, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при создании пользователя: " + err.Error(),
		})
		return
	}

	// Возвращаем данные созданного пользователя
	c.JSON(http.StatusCreated, models.RegisterResponse{
		ID:    id,
		Email: req.Email,
		Role:  req.Role,
	})
}
