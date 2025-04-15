package api

import (
	"pvz-service/internal/api/handlers"
	"pvz-service/internal/api/middleware"
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
	pvzQueries := queries.NewPVZQueries(db)
	receptionQueries := queries.NewReceptionQueries(db)
	productQueries := queries.NewProductQueries(db)

	newPasswordChecker := &utils.DefaultPasswordChecker{}

	// Создаем обработчики
	authHandler := handlers.NewAuthHandler(jwtManager, authQueries, newPasswordChecker)
	pvzHandler := handlers.NewPVZHandler(pvzQueries, receptionQueries, productQueries)
	receptionHandler := handlers.NewReceptionHandler(receptionQueries)
	productHandler := handlers.NewProductHandler(productQueries, receptionQueries)

	// Создаем middleware для авторизации
	authMiddleware := middleware.AuthMiddleware(jwtManager)
	requireModerator := middleware.RequireRole("moderator")

	// Публичные маршруты (без авторизации)
	publicRoutes := router.Group("")
	{
		// dummyLogin endpoint для получения тестового токена
		publicRoutes.POST("/dummyLogin", authHandler.DummyLogin)

		// Регистрация
		publicRoutes.POST("/register", authHandler.Register)

		// Вход
		publicRoutes.POST("/login", authHandler.Login)
	}

	// Защищенные маршруты (с авторизацией)
	protectedRoutes := router.Group("")
	protectedRoutes.Use(authMiddleware)

	protectedRoutes.POST("/receptions", authMiddleware, receptionHandler.CreateReception)

	protectedRoutes.POST("/products", productHandler.AddProduct)

	// Маршруты для работы с ПВЗ
	pvzRoutes := protectedRoutes.Group("/pvz")
	{
		// Создание ПВЗ (только для модераторов)
		pvzRoutes.POST("", requireModerator, pvzHandler.CreatePVZ)
		// Получение списка ПВЗ с фильтрацией и пагинацией
		pvzRoutes.GET("", pvzHandler.GetPVZList)

		pvzRoutes.POST("/:pvzId/close_last_reception", authMiddleware, receptionHandler.CloseLastReception)
		pvzRoutes.POST("/:pvzId/delete_last_product", productHandler.DeleteLastProduct)
	}

	return router
}
