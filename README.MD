# PVZ-service

A plants vs zombies... oh, more precisely pvz service for create,delete products in reception on PVZ.

## Основные моменты

Используемые технологии -> Docker, Docker-compose, PostgreSQL, Golang (squirrel (заместо ORM или SQLC), testify, Gin)

`migrations` - файл с миграциями для БД \
`internal` - внутренний код проекта

- `internal/tests` - Интеграционный тест, который
  Первым делом создает новый ПВЗ
  Добавляет новую приёмку заказов
  Добавляет 50 товаров в рамках текущей приёмки заказов
  Закрывает приёмку заказов

`cmd/server` - основной файл сервиса

Реализованы юнит-тесты для API бизнес логики.

## Требования

- Docker
- Docker Compose

## Быстрый старт

1. Клонируйте репозиторий любым удобным спосом, пример с https:

```bash
git clone https://github.com/falearn228/pvz-service.git
cd pvz-service
```

В **Makefile** описаны все возможные команды. \
в **app.config.env** установлены переменные окружения для подключения к БД и настройки токена.

2. Собираем, скачиваем контейнеры, перейдя в папку **avito-shop**

```bash
make build
```

3. Поднимаем контейнеры:

```bash
make up

# Проверяем, что все запустилось
docker-compose ps

# Остановка всех сервисов, при необходимости завершить работу
make down
```

4. Использование **API**:

---

## Аутентификация и пользователи

### 1. Получить тестовый токен (dummyLogin)

```bash
curl -X POST http://localhost:8080/dummyLogin \
     -H "Content-Type: application/json" \
     -d '{"role": "employee"}'
```

или

```bash
curl -X POST http://localhost:8080/dummyLogin \
     -H "Content-Type: application/json" \
     -d '{"role": "moderator"}'
```

### 2. Регистрация пользователя

```bash
curl -X POST http://localhost:8080/register \
     -H "Content-Type: application/json" \
     -d '{
           "email": "user@example.com",
           "password": "secure_password",
           "role": "employee"
         }'
```

### 3. Вход пользователя (login)

```bash
curl -X POST http://localhost:8080/login \
     -H "Content-Type: application/json" \
     -d '{
           "email": "user@example.com",
           "password": "secure_password"
         }'
```

---

## Работа с ПВЗ (Пунктами выдачи заказов)

### 4. Создать новый ПВЗ (только для moderator)

```bash
curl -X POST http://localhost:8080/pvz \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer " \
     -d '{"city": "Москва"}'
```

### 5. Получить список ПВЗ (фильтрация и пагинация)

```bash
curl -X GET "http://localhost:8080/pvz?startDate=2025-01-01T00:00:00Z&endDate=2025-12-31T23:59:59Z&page=1&limit=10" \
     -H "Authorization: Bearer "
```

Параметры:

- `startDate`, `endDate` — фильтрация по дате регистрации
- `page`, `limit` — пагинация

---

## Приёмки товаров

### 6. Создать новую приёмку (только для employee)

```bash
curl -X POST http://localhost:8080/receptions \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer " \
     -d '{"pvzId": ""}'
```

### 7. Закрыть последнюю открытую приёмку в ПВЗ (только для employee)

```bash
curl -X POST http://localhost:8080/pvz//close_last_reception \
     -H "Authorization: Bearer "
```

---

## Работа с товарами

### 8. Добавить товар в приёмку (только для employee)

```bash
curl -X POST http://localhost:8080/products \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer " \
     -d '{
           "type": "электроника",
           "pvzId": ""
         }'
```

Возможные значения поля `type`: `электроника`, `одежда`, `обувь`

### 9. Удалить последний добавленный товар из приёмки (только для employee)

```bash
curl -X POST http://localhost:8080/pvz//delete_last_product \
     -H "Authorization: Bearer "
```

---

## Примечания
- Все защищённые эндпоинты требуют заголовок `Authorization: Bearer `

---

5. Тестирование

```bash
# После старта контейнеров
make test
```
