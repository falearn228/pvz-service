package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	BaseURL = "http://localhost:8080"
)

// Структуры для запросов и ответов
type LoginRequest struct {
	Role string `json:"role"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type CreatePVZRequest struct {
	City string `json:"city"`
}

type PVZResponse struct {
	ID               string    `json:"id"`
	RegistrationDate time.Time `json:"registrationDate"`
	City             string    `json:"city"`
}

type CreateReceptionRequest struct {
	PvzID string `json:"pvzId"`
}

type ReceptionResponse struct {
	ID       string    `json:"id"`
	DateTime time.Time `json:"dateTime"`
	PvzID    string    `json:"pvzId"`
	Status   string    `json:"status"`
}

type CreateProductRequest struct {
	Type  string `json:"type"`
	PvzID string `json:"pvzId"`
}

type ProductResponse struct {
	ID          string    `json:"id"`
	DateTime    time.Time `json:"dateTime"`
	Type        string    `json:"type"`
	ReceptionID string    `json:"receptionId"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

// Функция для получения токена авторизации
func getAuthToken(t *testing.T, role string) string {
	loginData := LoginRequest{
		Role: role,
	}

	jsonData, err := json.Marshal(loginData)
	assert.NoError(t, err, "Ошибка при маршалинге данных для логина")

	req, err := http.NewRequest("POST", BaseURL+"/dummyLogin", bytes.NewBuffer(jsonData))
	assert.NoError(t, err, "Ошибка при создании запроса")

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Ошибка при выполнении запроса на логин")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Неверный статус-код при логине")

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Ошибка при чтении ответа")

	var loginResp LoginResponse
	err = json.Unmarshal(body, &loginResp)
	assert.NoError(t, err, "Ошибка при анмаршалинге ответа")

	return loginResp.Token
}

// Интеграционный тест
func TestPVZWorkflow(t *testing.T) {
	// Проверяем доступность сервера
	_, err := http.Get(BaseURL + "/dummyLogin")
	assert.NoError(t, err, "Сервер недоступен. Убедитесь, что сервер запущен")

	// Шаг 1: Создаём новый ПВЗ (нужен токен с ролью moderator)
	t.Log("1. Создание нового ПВЗ...")
	moderatorToken := getAuthToken(t, "moderator")

	pvzData := CreatePVZRequest{
		City: "Москва",
	}

	jsonData, err := json.Marshal(pvzData)
	assert.NoError(t, err, "Ошибка при маршалинге данных для создания ПВЗ")

	req, err := http.NewRequest("POST", BaseURL+"/pvz", bytes.NewBuffer(jsonData))
	assert.NoError(t, err, "Ошибка при создании запроса")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+moderatorToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Ошибка при выполнении запроса на создание ПВЗ")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Неверный статус-код при создании ПВЗ")

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Ошибка при чтении ответа")

	var pvzResp PVZResponse
	err = json.Unmarshal(body, &pvzResp)
	assert.NoError(t, err, "Ошибка при анмаршалинге ответа")

	pvzID := pvzResp.ID
	t.Logf("ПВЗ успешно создан, ID: %s", pvzID)

	// Получаем токен сотрудника для остальных операций
	employeeToken := getAuthToken(t, "employee")

	// Шаг 2: Добавляем новую приёмку
	t.Log("2. Создание новой приёмки заказов...")
	receptionData := CreateReceptionRequest{
		PvzID: pvzID,
	}

	jsonData, err = json.Marshal(receptionData)
	assert.NoError(t, err, "Ошибка при маршалинге данных для создания приёмки")

	req, err = http.NewRequest("POST", BaseURL+"/receptions", bytes.NewBuffer(jsonData))
	assert.NoError(t, err, "Ошибка при создании запроса")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+employeeToken)

	resp, err = client.Do(req)
	assert.NoError(t, err, "Ошибка при выполнении запроса на создание приёмки")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Неверный статус-код при создании приёмки")

	body, err = io.ReadAll(resp.Body)
	assert.NoError(t, err, "Ошибка при чтении ответа")

	var receptionResp ReceptionResponse
	err = json.Unmarshal(body, &receptionResp)
	assert.NoError(t, err, "Ошибка при анмаршалинге ответа")

	receptionID := receptionResp.ID
	t.Logf("Приёмка успешно создана, ID: %s", receptionID)

	// Шаг 3: Добавляем 50 товаров в приёмку
	t.Log("3. Добавление 50 товаров в приёмку...")
	productTypes := []string{"электроника", "одежда", "обувь"}

	for i := 0; i < 50; i++ {
		// Чередуем типы товаров для разнообразия
		productType := productTypes[i%len(productTypes)]

		productData := CreateProductRequest{
			Type:  productType,
			PvzID: pvzID,
		}

		jsonData, err = json.Marshal(productData)
		assert.NoError(t, err, "Ошибка при маршалинге данных для создания товара")

		req, err = http.NewRequest("POST", BaseURL+"/products", bytes.NewBuffer(jsonData))
		assert.NoError(t, err, "Ошибка при создании запроса")

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+employeeToken)

		resp, err = client.Do(req)
		assert.NoError(t, err, "Ошибка при выполнении запроса на создание товара")

		// Проверяем статус-код
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Ошибка при добавлении товара %d. Статус: %d, Ответ: %s",
				i+1, resp.StatusCode, string(body))
		}

		resp.Body.Close()

		// Выводим прогресс
		if (i+1)%10 == 0 {
			t.Logf("Добавлено %d товаров...", i+1)
		}

		// Небольшая задержка чтобы не перегружать сервер
		time.Sleep(100 * time.Millisecond)
	}

	t.Log("Все 50 товаров успешно добавлены")

	// Шаг 4: Закрываем приёмку
	t.Log("4. Закрытие приёмки заказов...")
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/pvz/%s/close_last_reception", BaseURL, pvzID), nil)
	assert.NoError(t, err, "Ошибка при создании запроса")

	req.Header.Set("Authorization", "Bearer "+employeeToken)

	resp, err = client.Do(req)
	assert.NoError(t, err, "Ошибка при выполнении запроса на закрытие приёмки")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Неверный статус-код при закрытии приёмки")

	body, err = io.ReadAll(resp.Body)
	assert.NoError(t, err, "Ошибка при чтении ответа")

	var closedReceptionResp ReceptionResponse
	err = json.Unmarshal(body, &closedReceptionResp)
	assert.NoError(t, err, "Ошибка при анмаршалинге ответа")

	t.Logf("Приёмка успешно закрыта, статус: %s", closedReceptionResp.Status)

	// Проверяем финальный статус
	assert.Equal(t, "close", closedReceptionResp.Status, "Неожиданный статус приёмки")

	t.Log("✅ Интеграционный тест успешно завершен!")
}
