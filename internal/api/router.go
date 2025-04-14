package api

import (
	"pvz-service/internal/api/handlers"
	"pvz-service/internal/config"
	"pvz-service/internal/db"
	"pvz-service/internal/db/queries"
	"pvz-service/internal/utils"

	"github.com/gin-gonic/gin"
)

func SetupRouter(config *config.Config, db *db.Database) *gin.Engine {
	// Создаем экземпляр Gin
	router := gin.Default()

	// Создаем менеджер JWT
	jwtManager := utils.NewJWTManager(&config.JWT)

	// Создаем запросы к базе данных
	authQueries := queries.NewAuthQueries(db)

	// Создаем обработчики
	authHandler := handlers.NewAuthHandler(jwtManager, authQueries)

	// Публичные маршруты (без авторизации)
	publicRoutes := router.Group("")
	{
		// dummyLogin endpoint для получения тестового токена
		publicRoutes.POST("/dummyLogin", authHandler.DummyLogin)

		// Регистрация
		publicRoutes.POST("/register", authHandler.Register)

		// Вход
		// publicRoutes.POST("/login", authHandler.Login)
	}

	return router
}
