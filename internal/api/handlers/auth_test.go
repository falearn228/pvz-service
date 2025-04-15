package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"pvz-service/internal/models"
	"pvz-service/internal/utils"
)

// Мок JWTManager
type MockJWTManager struct {
	mock.Mock
}

func (m *MockJWTManager) GenerateDummyToken(role string) (string, error) {
	args := m.Called(role)
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) GenerateToken(userID, role string) (string, error) {
	args := m.Called(userID, role)
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) ValidateToken(tokenString string) (*utils.CustomClaims, error) {
	args := m.Called(tokenString)
	return args.Get(0).(*utils.CustomClaims), args.Error(1)
}

// Мок AuthQueries
type MockAuthQueries struct {
	mock.Mock
}

func (m *MockAuthQueries) GetUserByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthQueries) CreateUser(ctx context.Context, email, passwordHash, role string) (string, error) {
	args := m.Called(ctx, email, passwordHash, role)
	return args.String(0), args.Error(1)
}

func (m *MockAuthQueries) GetUserWithCredentials(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

type MockPasswordChecker struct {
	mock.Mock
}

func (m *MockPasswordChecker) CheckPassword(password, hashedPassword string) error {
	args := m.Called(password, hashedPassword)
	return args.Error(0)
}

// Настройка тестового окружения
func setupAuthTest() (*gin.Engine, *MockJWTManager, *MockAuthQueries, *MockPasswordChecker) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	jwtManager := new(MockJWTManager)
	authQueries := new(MockAuthQueries)
	passwordChecker := new(MockPasswordChecker)

	authHandler := NewAuthHandler(jwtManager, authQueries, passwordChecker)

	r.POST("/dummyLogin", authHandler.DummyLogin)
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)

	return r, jwtManager, authQueries, passwordChecker
}

// TestDummyLoginSuccess проверяет успешный сценарий для dummyLogin
func TestDummyLoginSuccess(t *testing.T) {
	r, jwtManager, _, _ := setupAuthTest()

	// Настраиваем мок JWTManager для возврата токена
	jwtManager.On("GenerateDummyToken", "employee").Return("test-dummy-token", nil)

	// Создаем запрос с правильной структурой
	loginReq := models.DummyLoginRequest{
		Role: "employee",
	}
	jsonData, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/dummyLogin", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-dummy-token", response.Token)

	// Проверяем, что мок был вызван с правильными аргументами
	jwtManager.AssertExpectations(t)
}

// TestDummyLoginInvalidRole проверяет сценарий с некорректной ролью
func TestDummyLoginInvalidRole(t *testing.T) {
	r, _, _, _ := setupAuthTest()

	// Создаем запрос с некорректной ролью
	loginReq := map[string]string{
		"role": "invalid-role",
	}
	jsonData, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/dummyLogin", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должна быть ошибка валидации
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Неверный запрос")
}

// TestDummyLoginJWTError проверяет сценарий с ошибкой генерации JWT
func TestDummyLoginJWTError(t *testing.T) {
	r, jwtManager, _, _ := setupAuthTest()

	// Настраиваем мок JWTManager для возврата ошибки
	jwtManager.On("GenerateDummyToken", "moderator").Return("", errors.New("jwt generation error"))

	// Создаем запрос
	loginReq := models.DummyLoginRequest{
		Role: "moderator",
	}

	jsonData, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/dummyLogin", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должна быть внутренняя ошибка сервера
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка генерации токена")

	// Проверяем, что мок был вызван с правильными аргументами
	jwtManager.AssertExpectations(t)
}

// Продолжение файла internal/api/handlers/auth_test.go

// TestRegisterSuccess проверяет успешный сценарий регистрации
func TestRegisterSuccess(t *testing.T) {
	r, _, authQueries, _ := setupAuthTest()

	// Настраиваем моки
	authQueries.On("GetUserByEmail", mock.Anything, "new@example.com").Return(false, nil)
	authQueries.On("CreateUser", mock.Anything, "new@example.com", mock.AnythingOfType("string"), "employee").Return("test-uuid", nil)

	// Создаем запрос
	registerReq := models.RegisterRequest{
		Email:    "new@example.com",
		Password: "secure_password",
		Role:     "employee",
	}
	jsonData, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ
	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.RegisterResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-uuid", response.ID)
	assert.Equal(t, "new@example.com", response.Email)
	assert.Equal(t, "employee", response.Role)

	// Проверяем, что моки были вызваны с правильными аргументами
	authQueries.AssertExpectations(t)
}

// TestRegisterUserAlreadyExists проверяет сценарий с уже существующим пользователем
func TestRegisterUserAlreadyExists(t *testing.T) {
	r, _, authQueries, _ := setupAuthTest()

	// Настраиваем моки - пользователь уже существует
	authQueries.On("GetUserByEmail", mock.Anything, "existing@example.com").Return(true, nil)

	// Создаем запрос
	registerReq := models.RegisterRequest{
		Email:    "existing@example.com",
		Password: "secure_password",
		Role:     "employee",
	}
	jsonData, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должна быть ошибка о существующем пользователе
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Пользователь с таким email уже существует")

	// Проверяем, что моки были вызваны с правильными аргументами
	authQueries.AssertExpectations(t)
}

// TestRegisterInvalidData проверяет сценарий с некорректными данными
func TestRegisterInvalidData(t *testing.T) {
	r, _, _, _ := setupAuthTest()

	// Создаем запрос с некорректными данными
	registerReq := map[string]string{
		"email": "not-an-email",
		"role":  "employee",
		// отсутствует обязательное поле password
	}
	jsonData, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должна быть ошибка валидации
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Неверный запрос")
}

// TestRegisterDatabaseError проверяет сценарий с ошибкой базы данных
func TestRegisterDatabaseError(t *testing.T) {
	r, _, authQueries, _ := setupAuthTest()

	// Настраиваем моки - ошибка при проверке пользователя
	authQueries.On("GetUserByEmail", mock.Anything, "error@example.com").Return(false, errors.New("database error"))

	// Создаем запрос
	registerReq := models.RegisterRequest{
		Email:    "error@example.com",
		Password: "secure_password",
		Role:     "employee",
	}
	jsonData, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должна быть внутренняя ошибка сервера
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка при проверке email")

	// Проверяем, что моки были вызваны с правильными аргументами
	authQueries.AssertExpectations(t)
}

// TestLoginSuccess проверяет успешный вход в систему
func TestLoginSuccess(t *testing.T) {
	r, jwtManager, authQueries, passworcChecker := setupAuthTest()

	// Создаем тестового пользователя
	testUser := &models.User{
		ID:           "test-uuid",
		Email:        "user@example.com",
		Role:         "employee",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // хеш для пароля "password123"
	}

	// Настраиваем моки
	authQueries.On("GetUserWithCredentials", mock.Anything, "user@example.com").Return(testUser, nil)
	jwtManager.On("GenerateToken", "test-uuid", "employee").Return("test-token", nil)
	passworcChecker.On("CheckPassword", "password123", mock.Anything).Return(nil)

	// Создаем запрос
	loginReq := models.LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	}
	jsonData, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-token", response.Token)

	// Проверяем, что моки были вызваны с правильными аргументами
	authQueries.AssertExpectations(t)
	jwtManager.AssertExpectations(t)
}

// TestLoginUserNotFound проверяет сценарий с несуществующим пользователем
func TestLoginUserNotFound(t *testing.T) {
	r, _, authQueries, _ := setupAuthTest()

	// Настраиваем моки - пользователь не найден
	authQueries.On("GetUserWithCredentials", mock.Anything, "nonexistent@example.com").Return(nil, errors.New("user not found"))

	// Создаем запрос
	loginReq := models.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}
	jsonData, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Неверные учетные данные", response.Message)

	// Проверяем, что моки были вызваны с правильными аргументами
	authQueries.AssertExpectations(t)
}

// TestLoginInvalidPassword проверяет сценарий с неверным паролем
func TestLoginInvalidPassword(t *testing.T) {
	r, _, authQueries, passworcChecker := setupAuthTest()

	// Создаем тестового пользователя с хешем для пароля "password123"
	testUser := &models.User{
		ID:           "test-uuid",
		Email:        "user@example.com",
		Role:         "employee",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
	}

	// Настраиваем моки
	authQueries.On("GetUserWithCredentials", mock.Anything, "user@example.com").Return(testUser, nil)
	passworcChecker.On("CheckPassword", "wrong_password", testUser.PasswordHash).Return(errors.New("invalid password"))

	// Создаем запрос с неверным паролем
	loginReq := models.LoginRequest{
		Email:    "user@example.com",
		Password: "wrong_password",
	}
	jsonData, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Неверные учетные данные", response.Message)

	// Проверяем, что моки были вызваны с правильными аргументами
	authQueries.AssertExpectations(t)
}

// TestLoginTokenError проверяет сценарий с ошибкой генерации токена
func TestLoginTokenError(t *testing.T) {
	r, jwtManager, authQueries, passwordChecker := setupAuthTest()

	// Создаем тестового пользователя
	testUser := &models.User{
		ID:           "test-uuid",
		Email:        "user@example.com",
		Role:         "employee",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // хеш для пароля "password123"
	}

	// Настраиваем моки
	authQueries.On("GetUserWithCredentials", mock.Anything, "user@example.com").Return(testUser, nil)
	jwtManager.On("GenerateToken", "test-uuid", "employee").Return("", errors.New("token generation error"))
	passwordChecker.On("CheckPassword", "password123", testUser.PasswordHash).Return(nil)

	// Создаем запрос
	loginReq := models.LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	}
	jsonData, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 500 Internal Server Error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка при создании токена")

	// Проверяем, что моки были вызваны с правильными аргументами
	authQueries.AssertExpectations(t)
	jwtManager.AssertExpectations(t)
}

// TestLoginInvalidRequest проверяет сценарий с некорректными данными запроса
func TestLoginInvalidRequest(t *testing.T) {
	r, _, _, _ := setupAuthTest()

	// Создаем запрос с некорректными данными
	loginReq := map[string]string{
		"email": "not-an-email",
		// отсутствует обязательное поле password
	}
	jsonData, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должна быть ошибка валидации
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Неверный запрос")
}
