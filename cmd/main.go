// cmd/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Peranum/tg-dice/internal/databases"
	"github.com/Peranum/tg-dice/internal/databases/redis"
	applicationServices "github.com/Peranum/tg-dice/internal/user/application/services"
	domainServices "github.com/Peranum/tg-dice/internal/user/domain/services"
	userRepositories "github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
	userControllers "github.com/Peranum/tg-dice/internal/user/presentation/controllers"

	botServices "github.com/Peranum/tg-dice/internal/games/domain/bot/services"
	botRepositories "github.com/Peranum/tg-dice/internal/games/infrastructure/bot/repositories"
	botControllers "github.com/Peranum/tg-dice/internal/games/presentation/controllers/bot"

	referralServices "github.com/Peranum/tg-dice/internal/referral/domain/services"
	referralControllers "github.com/Peranum/tg-dice/internal/referral/presentation/controllers"

	slotServices "github.com/Peranum/tg-dice/internal/games/domain/slots/services"
	slotRepositories "github.com/Peranum/tg-dice/internal/games/infrastructure/slots/repositories"
	slotControllers "github.com/Peranum/tg-dice/internal/games/presentation/controllers/slots"

	presentation "github.com/Peranum/tg-dice/internal/games/presentation"

	historyServices "github.com/Peranum/tg-dice/internal/games/domain/history/services"
	historyRepositories "github.com/Peranum/tg-dice/internal/games/infrastructure/history/repositories"
	historyControllers "github.com/Peranum/tg-dice/internal/games/presentation/controllers/history/general"
	historyWebsockets "github.com/Peranum/tg-dice/internal/games/presentation/websockets/history"

	promoService "github.com/Peranum/tg-dice/internal/promocodes/domain/services"
	promoRepo "github.com/Peranum/tg-dice/internal/promocodes/infrastructure/repository"
	promoController "github.com/Peranum/tg-dice/internal/promocodes/presentation/controllers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "github.com/Peranum/tg-dice/docs" // подключение документации Swagger

	custommiddleware "github.com/Peranum/tg-dice/internal/middleware"
	testHandlers "github.com/Peranum/tg-dice/internal/test/handlers"
)

// @title TG-Dice API
// @version 1.0
// @description API for managing users and games in TG-Dice application
// @host api.m5dice.com
// @BasePath /
// @schemes https
func main() {

	// Получение значений из переменных окружения
	mongoURI := os.Getenv("MONGO_URI")
	dbName := os.Getenv("DB_NAME")
	port := os.Getenv("PORT")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatalf("TELEGRAM_BOT_TOKEN не задан!")
	}

	if mongoURI == "" || dbName == "" || port == "" || redisHost == "" || redisPort == "" || redisPassword == "" {
		log.Fatalf("Не все переменные окружения заданы!")
	}

	// Инициализация MongoDB
	db, err := databases.ConnectDB(mongoURI, dbName)
	if err != nil {
		log.Fatalf("Не удалось подключиться к MongoDB: %v", err)
	}
	defer func() {
		if err := db.Client().Disconnect(context.Background()); err != nil {
			log.Fatalf("Ошибка при отключении от MongoDB: %v", err)
		}
	}()

	// Инициализация Redis
	redis.InitRedis(redisHost, redisPort, redisPassword)

	// Создаем конфигурацию для middleware
	telegramAuthConfig := custommiddleware.TelegramAuthConfig{
		BotToken: botToken,
	}

	// Репозитории и сервисы для пользователей
	userRepo := userRepositories.NewUserRepository(db)
	referralService := referralServices.NewReferralService(userRepo)
	userDomainService := domainServices.NewUserDomainService(userRepo, referralService)
	withdrawalsRepo := userRepositories.NewWithdrawalsRepository(db)
	withdrawalService := domainServices.NewWithdrawalService(withdrawalsRepo, userRepo)
	userAppService := applicationServices.NewUserAppService(userDomainService, withdrawalService)
	userController := userControllers.NewUserController(userAppService)

	// Репозитории и сервисы для реферальной системы
	referralController := referralControllers.NewReferralController(referralService)
	historyRepo := historyRepositories.NewGameRepository(db)
	websocketServer := historyWebsockets.NewWebSocketServer()
	historyService := historyServices.NewGameService(historyRepo, websocketServer)
	historyController := historyControllers.NewGameHistoryController(historyService)

	// Репозитории и сервисы для игры с ботом
	botRepo := botRepositories.NewBotRepository(db)
	botGameService := botServices.NewBotGameService(botRepo, userRepo, historyService, referralService)
	botGameController := botControllers.NewBotGameController(botGameService)

	// Репозитории и сервисы для слотов
	slotBalanceRepo := slotRepositories.NewSlotsBalanceRepository(db)
	slotGameRepo := slotRepositories.NewSlotGameRepository(db)
	slotGameService := slotServices.NewSlotGameService(slotGameRepo, userRepo, slotBalanceRepo)
	slotsBalanceService := slotServices.NewSlotsBalanceService(slotBalanceRepo)
	slotGameController := slotControllers.NewSlotGameController(slotGameService, slotsBalanceService)

	// Репозиторий для промокодов
	promoCodeRepo := promoRepo.NewPromoCodeRepository(db, userRepo)
	promoCodeService := promoService.NewPromoCodeService(promoCodeRepo, userRepo)
	promoCodeController := promoController.NewPromoCodeController(promoCodeService)

	// Инициализация Echo
	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://127.0.0.1", "https://127.0.0.1", "http://localhost", "https://localhost", "https://m5dice.com", "https://www.m5dice.com", "https://webassist.ngrok.dev", "https://www.webassist.ngrok.dev", "http://38.180.244.162", "http://85.235.150.22"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{"Content-Type", "Authorization", "X-Requested-With", "Accept", "Origin"},
	}))

	// Добавляем Telegram Auth middleware
	e.Use(custommiddleware.TelegramAuthMiddleware(telegramAuthConfig))

	// Swagger
	e.Static("/docs", "./docs")
	e.GET("/swagger/*", echoSwagger.WrapHandler, middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == "admin" && password == "username" {
			return true, nil
		}
		return false, nil
	}))
	// Роуты для пользователей
	e.POST("/users", userController.CreateUser)
	e.GET("/users/:id", userController.GetUser)
	e.PATCH("/users/tgid/:tgid", userController.PatchUserByTgID)
	e.DELETE("/users/:id", userController.DeleteUser)
	e.GET("/users", userController.ListUsers)
	e.GET("/users/:wallet/balances", userController.GetUserBalances)
	e.GET("/users/:wallet/referral-code", userController.GetReferralCodeHandler)
	e.GET("/users/name/:name", userController.GetUserByName)
	e.GET("/users/:wallet/referral-earnings", userController.GetReferralEarnings)
	e.GET("/users/:wallet/points", userController.GetUserPointsByWallet) // Получение очков пользователя по кошельку
	e.GET("/users/points", userController.GetUsersSortedByPoints)
	e.GET("/users/withdrawal/:id", userController.GetWithdrawal)
	e.GET("/users/:wallet/withdrawals", userController.GetWithdrawalsByWallet)
	e.GET("/users/withdrawals/last-50", userController.GetLast50Withdrawals)
	e.GET("/users/withdrawals/last-50-with-jetton", userController.GetLast50WithdrawalsWithJetton)
	e.GET("/users/withdrawals/last-50-without-jetton", userController.GetLast50WithdrawalsWithoutJetton)
	e.DELETE("/users/withdrawal/:id", userController.DeleteWithdrawal)
	e.POST("/withdrawals", userController.CreateWithdrawal)

	e.GET("/referrals/level", referralController.GetReferralsByLevelHandler)
	e.GET("/referrals/total", referralController.GetTotalReferralsHandler)
	e.GET("/referrals/levels", referralController.GetReferralsByLevelsHandler)

	// Роут для игры в кости
	e.POST("/games/dice", botGameController.PlayDiceGameHandler)
	e.POST("/games/simulate-user-win/:wallet", botGameController.SimulateUserWinHandler)
	e.POST("/games/bot/balance", botGameController.InitializeBotBalanceHandler)
	e.GET("/bot/balance/:tokenType", botGameController.GetSpecificTokenBalance)
	e.GET("/bot/balance", botGameController.GetBotBalance)
	e.POST("/bot/balance/add", botGameController.AddTokensToBotBalanceHandler)
	e.POST("/bot/balance/subtract", botGameController.SubtractTokensFromBotBalanceHandler)

	// Роуты для слотов
	e.POST("/slots/play", slotGameController.PlaySlot)
	e.GET("/slots/:wallet/games", slotGameController.GetGamesByWallet)
	e.GET("/slots/:wallet/recent-games", slotGameController.GetRecentGames)
	e.GET("/slots/history", slotGameController.GetHistory)
	e.POST("/slots/balance/initialize", slotGameController.InitializeBalance) // Инициализация баланса
	e.GET("/slots/balance", slotGameController.GetBalance)                    // Получение текущего баланса
	// Роуты для управления токенами в слотах
	e.POST("/slots/balance/add", slotGameController.AddTokens)           // Добавление токенов
	e.POST("/slots/balance/subtract", slotGameController.SubtractTokens) // Вычитание токенов

	e.POST("/games/history", historyController.SaveGame)
	e.GET("/games/history", historyController.GetGamesHistory)            // Получение общей истории
	e.GET("/games/history/:wallet", historyController.GetUserGameHistory) // Получение истории для конкретного пользователя

	e.POST("/promocodes/create", promoCodeController.CreatePromoCode)
	e.POST("/promocodes/activate", promoCodeController.ActivatePromoCode)
	e.GET("/promocodes/active", promoCodeController.ListActivePromoCodes)
	e.GET("/promocodes/:code", promoCodeController.GetPromoCode)
	e.POST("/promocodes/expire", promoCodeController.ExpirePromoCodes)

	// Инициализация тестовых обработчиков
	testHandler := testHandlers.NewTestHandler(botToken)

	// Регистрация тестовых маршрутов
	e.GET("/test", testHandler.GenerateTestQuery)
	e.POST("/test_post", testHandler.ValidateTestQuery)

	// Инициализация сервиса PvP игр
	pvpService := presentation.NewDicePVPGameService(userRepo, historyService)

	// Добавляем маршруты для WebSocket
	e.GET("/ws/dice", func(c echo.Context) error {
		pvpService.HandleWebSocket(c.Response(), c.Request())
		return nil
	})

	e.GET("/ws/history", func(c echo.Context) error {
		websocketServer.HandleConnection(c.Response(), c.Request())
		return nil
	})

	// Запуск сервера
	log.Printf("Запуск сервера на порту %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}

}
