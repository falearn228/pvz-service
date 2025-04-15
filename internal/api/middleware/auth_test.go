package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"pvz-service/internal/models"
	"pvz-service/internal/utils"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockJWTManager мокирует JWTManager для тестирования
type MockJWTManager struct {
	mock.Mock
}

// ValidateToken мокирует проверку токена
func (m *MockJWTManager) ValidateToken(tokenString string) (*utils.CustomClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*utils.CustomClaims), args.Error(1)
}

func (m *MockJWTManager) GenerateDummyToken(role string) (string, error) {
	args := m.Called(role)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) GenerateToken(userID, role string) (string, error) {
	args := m.Called(userID, role)
	if args.Get(0) == nil || args.Get(1) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

// setupAuthTest настраивает тестовое окружение
func setupAuthTest() (*gin.Engine, *MockJWTManager) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	jwtManager := new(MockJWTManager)

	return r, jwtManager
}

// TestAuthMiddlewareValidToken проверяет успешную авторизацию с валидным токеном
func TestAuthMiddlewareValidToken(t *testing.T) {
	r, jwtManager := setupAuthTest()

	// Создаем тестовые данные
	validToken := "valid.jwt.token"
	claims := &utils.CustomClaims{
		UserID: "user123",
		Role:   "employee",
	}

	// Настраиваем мок
	jwtManager.On("ValidateToken", validToken).Return(claims, nil)

	// Настраиваем маршрут с middleware
	r.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
		// Проверяем, что данные пользователя сохранены в контексте
		userID, exists := c.Get("userID")
		assert.True(t, exists)
		assert.Equal(t, "user123", userID)

		userRole, exists := c.Get("userRole")
		assert.True(t, exists)
		assert.Equal(t, "employee", userRole)

		c.Status(http.StatusOK)
	})

	// Создаем запрос
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что мок был вызван с правильными аргументами
	jwtManager.AssertExpectations(t)
}

// TestAuthMiddlewareMissingToken проверяет случай с отсутствующим токеном
func TestAuthMiddlewareMissingToken(t *testing.T) {
	r, jwtManager := setupAuthTest()

	// Настраиваем маршрут с middleware
	r.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
		// Этот обработчик не должен быть вызван
		t.Fail()
	})

	// Создаем запрос без токена
	req, _ := http.NewRequest("GET", "/protected", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Проверяем сообщение об ошибке
	var response models.ErrorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "Отсутствует токен авторизации", response.Message)

	// Проверяем, что мок не был вызван
	jwtManager.AssertNotCalled(t, "ValidateToken")
}

// TestAuthMiddlewareInvalidTokenFormat проверяет случай с неверным форматом токена
func TestAuthMiddlewareInvalidTokenFormat(t *testing.T) {
	r, jwtManager := setupAuthTest()

	// Настраиваем маршрут с middleware
	r.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
		// Этот обработчик не должен быть вызван
		t.Fail()
	})

	// Тест 1: Неверный префикс
	req1, _ := http.NewRequest("GET", "/protected", nil)
	req1.Header.Set("Authorization", "Token abc123")

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusUnauthorized, w1.Code)

	var response1 models.ErrorResponse
	assert.NoError(t, json.Unmarshal(w1.Body.Bytes(), &response1))
	assert.Equal(t, "Неверный формат токена", response1.Message)

	// Тест 2: Отсутствие частей
	req2, _ := http.NewRequest("GET", "/protected", nil)
	req2.Header.Set("Authorization", "Bearer")

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	var response2 models.ErrorResponse
	assert.NoError(t, json.Unmarshal(w2.Body.Bytes(), &response2))
	assert.Equal(t, "Неверный формат токена", response2.Message)

	// Проверяем, что мок не был вызван
	jwtManager.AssertNotCalled(t, "ValidateToken")
}

// TestAuthMiddlewareInvalidToken проверяет случай с недействительным токеном
func TestAuthMiddlewareInvalidToken(t *testing.T) {
	r, jwtManager := setupAuthTest()

	// Создаем тестовые данные
	invalidToken := "invalid.jwt.token"

	// Настраиваем мок
	jwtManager.On("ValidateToken", invalidToken).Return(nil, errors.New("token has expired"))

	// Настраиваем маршрут с middleware
	r.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
		// Этот обработчик не должен быть вызван
		t.Fail()
	})

	// Создаем запрос
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+invalidToken)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Проверяем сообщение об ошибке
	var response models.ErrorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "Неверный токен: token has expired", response.Message)

	// Проверяем, что мок был вызван с правильными аргументами
	jwtManager.AssertExpectations(t)
}

// TestRequireRoleAuthorized проверяет успешную авторизацию с правильной ролью
func TestRequireRoleAuthorized(t *testing.T) {
	r, _ := setupAuthTest()

	// Настраиваем маршрут с middleware
	r.GET("/admin", RequireRole("admin"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Создаем запрос
	req, _ := http.NewRequest("GET", "/admin", nil)

	// Создаем тестовый контекст с установленной ролью
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Set("userRole", "admin")

	// Вызываем middleware напрямую
	RequireRole("admin")(ctx)

	// Проверяем, что выполнение не было прервано
	assert.False(t, ctx.IsAborted())
}

// TestRequireRoleForbidden проверяет запрет доступа с неправильной ролью
func TestRequireRoleForbidden(t *testing.T) {
	r, _ := setupAuthTest()

	// Настраиваем маршрут с middleware
	r.GET("/admin", RequireRole("admin"), func(c *gin.Context) {
		// Этот обработчик не должен быть вызван
		t.Fail()
	})

	// Создаем запрос
	req, _ := http.NewRequest("GET", "/admin", nil)

	// Создаем тестовый контекст с установленной ролью
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Set("userRole", "employee")

	// Вызываем middleware напрямую
	RequireRole("admin")(ctx)

	// Проверяем, что выполнение было прервано
	assert.True(t, ctx.IsAborted())
	assert.Equal(t, http.StatusForbidden, w.Code)

	// Проверяем сообщение об ошибке
	var response models.ErrorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "Доступ запрещен: недостаточно прав", response.Message)
}

// TestRequireRoleNoUser проверяет случай с отсутствием данных о пользователе
func TestRequireRoleNoUser(t *testing.T) {
	r, _ := setupAuthTest()

	// Настраиваем маршрут с middleware
	r.GET("/admin", RequireRole("admin"), func(c *gin.Context) {
		// Этот обработчик не должен быть вызван
		t.Fail()
	})

	// Создаем запрос
	req, _ := http.NewRequest("GET", "/admin", nil)

	// Создаем тестовый контекст без установленной роли
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	// Вызываем middleware напрямую
	RequireRole("admin")(ctx)

	// Проверяем, что выполнение было прервано
	assert.True(t, ctx.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Проверяем сообщение об ошибке
	var response models.ErrorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "Нет данных о пользователе", response.Message)
}

// TestAuthMiddlewareWithRequireRole проверяет совместную работу обоих middleware
func TestAuthMiddlewareWithRequireRole(t *testing.T) {
	r, jwtManager := setupAuthTest()

	// Создаем тестовые данные
	validToken := "valid.jwt.token"
	claims := &utils.CustomClaims{
		UserID: "user123",
		Role:   "admin",
	}

	// Настраиваем мок
	jwtManager.On("ValidateToken", validToken).Return(claims, nil)

	// Настраиваем маршрут с обоими middleware
	r.GET("/admin", AuthMiddleware(jwtManager), RequireRole("admin"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Создаем запрос с валидным токеном
	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что мок был вызван с правильными аргументами
	jwtManager.AssertExpectations(t)

	// Тест с неправильной ролью
	claims.Role = "employee"
	jwtManager.On("ValidateToken", validToken).Return(claims, nil)

	// Создаем новый запрос
	req2, _ := http.NewRequest("GET", "/admin", nil)
	req2.Header.Set("Authorization", "Bearer "+validToken)

	// Выполняем запрос
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	// Проверяем ответ - должен быть статус 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w2.Code)

	// Проверяем сообщение об ошибке
	var response models.ErrorResponse
	assert.NoError(t, json.Unmarshal(w2.Body.Bytes(), &response))
	assert.Equal(t, "Доступ запрещен: недостаточно прав", response.Message)
}
