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
	jwtManager      utils.JWTManagerInterface
	authQueries     queries.AuthQueriesInterface
	passwordChecker utils.PasswordCheckerInterface
}

// NewAuthHandler создает новый экземпляр AuthHandler
func NewAuthHandler(jwtManager utils.JWTManagerInterface, authQueries queries.AuthQueriesInterface, passwordChecker utils.PasswordCheckerInterface) *AuthHandler {
	return &AuthHandler{
		jwtManager:      jwtManager,
		authQueries:     authQueries,
		passwordChecker: passwordChecker,
	}
}

// DummyLogin обрабатывает запрос на получение тестового токена
func (h *AuthHandler) DummyLogin(c *gin.Context) {
	var req models.DummyLoginRequest

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
	c.JSON(http.StatusOK, models.DummyLoginResponse{
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

// Login обрабатывает запрос на авторизацию пользователя
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	// Проверяем запрос
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Неверный запрос: " + err.Error(),
		})
		return
	}

	// Получаем пользователя из базы данных
	user, err := h.authQueries.GetUserWithCredentials(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Message: "Неверные учетные данные",
		})
		return
	}

	// Проверяем пароль - используем PasswordHash
	err = h.passwordChecker.CheckPassword(req.Password, user.PasswordHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Message: "Неверные учетные данные",
		})
		return
	}

	// Генерируем JWT-токен
	token, err := h.jwtManager.GenerateToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при создании токена: " + err.Error(),
		})
		return
	}

	// Возвращаем токен
	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
	})
}
