package queries

import (
	"context"
	"database/sql"
	"errors"
	"pvz-service/internal/db"
	"pvz-service/internal/models"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// Тогда в тестах создаем экземпляр db.Database:
func setupAuthQueriesTest(t *testing.T) (*AuthQueries, sqlmock.Sqlmock) {
	mockDB, mock, _ := sqlmock.New()
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	// Оборачиваем в структуру Database из пакета db
	dbInstance := &db.Database{DB: sqlxDB}

	q := &AuthQueries{
		db: dbInstance,
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	return q, mock
}

func TestCreateUser(t *testing.T) {
	testCases := []struct {
		name         string
		email        string
		passwordHash string
		role         string
		mockSetup    func(mock sqlmock.Sqlmock)
		expectedID   string
		expectedErr  bool
	}{
		{
			name:         "Успешное создание пользователя",
			email:        "user@example.com",
			passwordHash: "hash123",
			role:         "employee",
			mockSetup: func(mock sqlmock.Sqlmock) {
				expectedSQL := `INSERT INTO users \(email,password_hash,role,created_at\) VALUES \(\$1,\$2,\$3,CURRENT_TIMESTAMP\) RETURNING id`
				mock.ExpectQuery(expectedSQL).
					WithArgs("user@example.com", "hash123", "employee").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("123e4567-e89b-12d3-a456-426614174000"))
			},
			expectedID:  "123e4567-e89b-12d3-a456-426614174000",
			expectedErr: false,
		},
		{
			name:         "Ошибка базы данных",
			email:        "user@example.com",
			passwordHash: "hash123",
			role:         "employee",
			mockSetup: func(mock sqlmock.Sqlmock) {
				expectedSQL := `INSERT INTO users \(email,password_hash,role,created_at\) VALUES \(\$1,\$2,\$3,CURRENT_TIMESTAMP\) RETURNING id`
				mock.ExpectQuery(expectedSQL).
					WithArgs("user@example.com", "hash123", "employee").
					WillReturnError(errors.New("database error"))
			},
			expectedID:  "",
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Настройка
			q, mock := setupAuthQueriesTest(t)
			tc.mockSetup(mock)

			// Выполнение
			id, err := q.CreateUser(context.Background(), tc.email, tc.passwordHash, tc.role)

			// Проверка
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, id)
			}

			// Проверка, что все ожидания были выполнены
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Остались невыполненные ожидания: %s", err)
			}
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	testCases := []struct {
		name        string
		email       string
		mockSetup   func(mock sqlmock.Sqlmock)
		expected    bool
		expectedErr bool
	}{
		{
			name:  "Пользователь существует",
			email: "existing@example.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				expectedSQL := `SELECT 1 FROM users WHERE email = \$1 LIMIT 1`
				mock.ExpectQuery(expectedSQL).
					WithArgs("existing@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
			},
			expected:    true,
			expectedErr: false,
		},
		{
			name:  "Пользователь не существует",
			email: "nonexisting@example.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				expectedSQL := `SELECT 1 FROM users WHERE email = \$1 LIMIT 1`
				mock.ExpectQuery(expectedSQL).
					WithArgs("nonexisting@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    false,
			expectedErr: false,
		},
		{
			name:  "Ошибка базы данных",
			email: "error@example.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				expectedSQL := `SELECT 1 FROM users WHERE email = \$1 LIMIT 1`
				mock.ExpectQuery(expectedSQL).
					WithArgs("error@example.com").
					WillReturnError(errors.New("database error"))
			},
			expected:    false,
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Настройка
			q, mock := setupAuthQueriesTest(t)
			tc.mockSetup(mock)

			// Выполнение
			exists, err := q.GetUserByEmail(context.Background(), tc.email)

			// Проверка
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, exists)
			}

			// Проверка, что все ожидания были выполнены
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Остались невыполненные ожидания: %s", err)
			}
		})
	}
}

func TestGetUserWithCredentials(t *testing.T) {
	testCases := []struct {
		name        string
		email       string
		mockSetup   func(mock sqlmock.Sqlmock)
		expected    *models.User
		expectedErr bool
	}{
		{
			name:  "Успешное получение пользователя",
			email: "user@example.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				expectedSQL := `SELECT id, email, role, password_hash FROM users WHERE email = \$1 LIMIT 1`
				mock.ExpectQuery(expectedSQL).
					WithArgs("user@example.com").
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "email", "role", "password_hash"}).
							AddRow("123e4567-e89b-12d3-a456-426614174000", "user@example.com", "employee", "hash123"),
					)
			},
			expected: &models.User{
				ID:           "123e4567-e89b-12d3-a456-426614174000",
				Email:        "user@example.com",
				Role:         "employee",
				PasswordHash: "hash123",
			},
			expectedErr: false,
		},
		{
			name:  "Пользователь не найден",
			email: "notfound@example.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				expectedSQL := `SELECT id, email, role, password_hash FROM users WHERE email = \$1 LIMIT 1`
				mock.ExpectQuery(expectedSQL).
					WithArgs("notfound@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name:  "Ошибка базы данных",
			email: "error@example.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				expectedSQL := `SELECT id, email, role, password_hash FROM users WHERE email = \$1 LIMIT 1`
				mock.ExpectQuery(expectedSQL).
					WithArgs("error@example.com").
					WillReturnError(errors.New("database error"))
			},
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Настройка
			q, mock := setupAuthQueriesTest(t)
			tc.mockSetup(mock)

			// Выполнение
			user, err := q.GetUserWithCredentials(context.Background(), tc.email)

			// Проверка
			if tc.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected.ID, user.ID)
				assert.Equal(t, tc.expected.Email, user.Email)
				assert.Equal(t, tc.expected.Role, user.Role)
				assert.Equal(t, tc.expected.PasswordHash, user.PasswordHash)
			}

			// Проверка, что все ожидания были выполнены
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Остались невыполненные ожидания: %s", err)
			}
		})
	}
}
