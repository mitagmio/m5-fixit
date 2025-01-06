package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Peranum/tg-dice/internal/games/domain/bot/services"
	"github.com/labstack/echo/v4"
)

// PlayDiceGameRequest структура для данных из тела запроса
type PlayDiceGameRequest struct {
	Wallet      string  `json:"wallet" validate:"required"`                     // Адрес кошелька
	TokenType   string  `json:"token_type" validate:"required"`                 // Тип токена
	BetAmount   float64 `json:"bet_amount" validate:"required,gt=0"`            // Ставка
	TargetScore int     `json:"target_score" validate:"required,gte=15,lte=45"` // Цель по очкам
}

type BotGameController struct {
	GameService *services.BotGameService
}

func NewBotGameController(gameService *services.BotGameService) *BotGameController {
	return &BotGameController{
		GameService: gameService,
	}
}

// PlayDiceGameHandler обрабатывает запрос на игру в кости с ботом
// @Summary Play Dice Game with Bot
// @Description Play a dice game with the bot, returning detailed round-by-round results
// @Tags bot, games
// @Accept json
// @Produce json
// @Param data body PlayDiceGameRequest true "Game data"  // Правильная аннотация для параметра body
// @Success 200 {object} map[string]interface{} "Игровой результат"
// @Failure 400 {object} map[string]string "Ошибка с параметрами запроса"
// @Failure 500 {object} map[string]string "Ошибка при обработке запроса"
// @Router /games/dice [post]
func (c *BotGameController) PlayDiceGameHandler(ctx echo.Context) error {
	// Читаем данные из тела запроса
	var request PlayDiceGameRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// Валидация данных
	if request.Wallet == "" || request.TokenType == "" || request.BetAmount <= 0 || request.TargetScore < 15 || request.TargetScore > 45 {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request parameters"})
	}

	// Логирование входных данных
	log.Printf("[PlayDiceGameHandler] Received request: wallet=%s, token_type=%s, bet_amount=%f, target_score=%d",
		request.Wallet, request.TokenType, request.BetAmount, request.TargetScore)

	// Вызов сервиса для игры
	result, err := c.GameService.PlayDiceGame(ctx.Request().Context(), request.Wallet, request.TokenType, request.BetAmount, request.TargetScore)
	if err != nil {
		log.Printf("[PlayDiceGameHandler] Error: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Возвращаем результат (result содержит все раунды и победителя)
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"result": result,
	})
}

// InitializeBotBalanceRequest структура для данных из тела запроса на создание баланса
type InitializeBotBalanceRequest struct {
	TonBalance float64 `json:"ton_balance" validate:"required,gt=0"` // Баланс Ton
	M5Balance  float64 `json:"m5_balance" validate:"required,gt=0"`  // Баланс M5
	DfcBalance float64 `json:"dfc_balance" validate:"required,gt=0"` // Баланс DFC
}

// InitializeBotBalanceHandler обрабатывает запрос на создание баланса бота
// @Summary Initialize Bot Balance
// @Description Initialize bot balance in the system
// @Tags bot
// @Accept json
// @Produce json
// @Param data body InitializeBotBalanceRequest true "Bot balance data"  // Параметры баланса бота
// @Success 201 {object} map[string]string "Баланс бота успешно инициализирован"
// @Failure 400 {object} map[string]string "Ошибка с параметрами запроса"
// @Failure 500 {object} map[string]string "Ошибка при обработке запроса"
// @Router /games/bot/balance [post]
func (c *BotGameController) InitializeBotBalanceHandler(ctx echo.Context) error {
	// Читаем данные из тела запроса
	var request InitializeBotBalanceRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// Валидация данных
	if request.TonBalance <= 0 || request.M5Balance <= 0 || request.DfcBalance <= 0 {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid balance values"})
	}

	// Логирование входных данных
	log.Printf("Received request to initialize bot balance: ton_balance=%f, m5_balance=%f, dfc_balance=%f",
		request.TonBalance, request.M5Balance, request.DfcBalance)

	// Вызов сервиса для создания баланса
	err := c.GameService.InitializeBotBalance(ctx.Request().Context(), request.TonBalance, request.M5Balance, request.DfcBalance)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Возвращаем успешный ответ
	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Bot balance initialized successfully"})
}

// SimulateUserWinHandler godoc
// @Summary Симуляция победы пользователя в игре
// @Description Симулирует игру, где пользователь выигрывает. Результат игры и начисление кубов.
// @Tags games
// @Accept json
// @Produce json
// @Param wallet path string true "Wallet адрес пользователя"
// @Success 200 {object} map[string]string "result: game result"
// @Failure 400 {object} map[string]string "error: wallet is required"
// @Failure 500 {object} map[string]string "error: error message"
// @Router /games/simulate-user-win/{wallet} [post]
func (gc *BotGameController) SimulateUserWinHandler(c echo.Context) error {
	// Получаем параметр wallet из URL
	wallet := c.Param("wallet")
	if wallet == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "wallet is required"})
	}

	// Вызываем метод сервиса для симуляции победы пользователя
	result, err := gc.GameService.SimulateDiceGameForUserWin(c.Request().Context(), wallet)
	if err != nil {
		log.Printf("[SimulateUserWinHandler] Failed to simulate game: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Возвращаем результат игры в ответе
	return c.JSON(http.StatusOK, map[string]string{
		"result": result,
	})
}

// GetBotBalance возвращает все балансы бота
// @Summary Получает все балансы бота
// @Description Возвращает все балансы бота
// @Tags bot-balance
// @Accept  json
// @Produce  json
// @Success 200 {object} entities.BotBalanceEntity
// @Failure 500 {string} string "Ошибка при получении баланса бота"
// @Router /bot/balance [get]
func (c *BotGameController) GetBotBalance(ctx echo.Context) error {
	balance, err := c.GameService.GetBotBalance(ctx.Request().Context())
	if err != nil {
		log.Printf("Ошибка при получении баланса бота: %v", err)
		return ctx.JSON(http.StatusInternalServerError, "Ошибка при получении баланса бота")
	}

	return ctx.JSON(http.StatusOK, balance)
}

// GetSpecificTokenBalance возвращает баланс по конкретному токену
// @Summary Получает баланс конкретного токена
// @Description Возвращает баланс конкретного токена
// @Tags bot-balance
// @Accept  json
// @Produce  json
// @Param tokenType path string true "Тип токена (ton_balance, m5_balance, dfc_balance)"
// @Success 200 {number} float64
// @Failure 400 {string} string "Некорректный тип токена"
// @Failure 500 {string} string "Ошибка при получении баланса токена"
// @Router /bot/balance/{tokenType} [get]
func (c *BotGameController) GetSpecificTokenBalance(ctx echo.Context) error {
	tokenType := ctx.Param("tokenType")
	balance, err := c.GameService.GetTokenBalance(ctx.Request().Context(), tokenType)
	if err != nil {
		if err.Error() == "invalid token type" {
			return ctx.JSON(http.StatusBadRequest, err.Error())
		}
		log.Printf("Ошибка при получении баланса токена %s: %v", tokenType, err)
		return ctx.JSON(http.StatusInternalServerError, "Ошибка при получении баланса токена")
	}

	return ctx.JSON(http.StatusOK, balance)
}

// AddTokensToBotBalanceRequest структура для данных из тела запроса
type AddTokensToBotBalanceRequest struct {
	TokenType string  `json:"token_type" validate:"required"`  // Тип токена (например, ton_balance)
	Amount    float64 `json:"amount" validate:"required,gt=0"` // Сумма для добавления
}

// AddTokensToBotBalanceHandler обрабатывает запрос на добавление токенов к балансу бота
// @Summary Add tokens to bot balance
// @Description Adds the specified amount of tokens to the bot balance
// @Tags bot-balance
// @Accept json
// @Produce json
// @Param data body AddTokensToBotBalanceRequest true "Token type and amount to add"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Validation error or invalid parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /bot/balance/add [post]
func (c *BotGameController) AddTokensToBotBalanceHandler(ctx echo.Context) error {
	// Читаем данные из тела запроса
	var request AddTokensToBotBalanceRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// Валидация данных
	if request.TokenType == "" || request.Amount <= 0 {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid token type or amount"})
	}

	// Логирование входных данных
	log.Printf("Received request to add tokens to bot balance: token_type=%s, amount=%f",
		request.TokenType, request.Amount)

	// Вызов сервиса для добавления токенов к балансу бота
	err := c.GameService.AddTokensToBotBalance(ctx.Request().Context(), request.TokenType, request.Amount)
	if err != nil {
		if err.Error() == "invalid token type" {
			return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		log.Printf("Ошибка при добавлении токенов к балансу бота: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add tokens to bot balance"})
	}

	// Возвращаем успешный ответ
	return ctx.JSON(http.StatusOK, map[string]string{"message": "Tokens added to bot balance successfully"})
}

// SubtractTokensFromBotBalanceRequest структура для данных из тела запроса
type SubtractTokensFromBotBalanceRequest struct {
	TokenType string  `json:"token_type" validate:"required"`  // Тип токена (например, ton_balance)
	Amount    float64 `json:"amount" validate:"required,gt=0"` // Сумма для вычитания
}

// SubtractTokensFromBotBalanceHandler обрабатывает запрос на вычитание токенов из баланса бота
// @Summary Subtract tokens from bot balance
// @Description Subtracts the specified amount of tokens from the bot balance
// @Tags bot-balance
// @Accept json
// @Produce json
// @Param data body SubtractTokensFromBotBalanceRequest true "Token type and amount to subtract"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Validation error or invalid parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /bot/balance/subtract [post]
func (c *BotGameController) SubtractTokensFromBotBalanceHandler(ctx echo.Context) error {
	// Читаем данные из тела запроса
	var request SubtractTokensFromBotBalanceRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// Валидация данных
	if request.TokenType == "" || request.Amount <= 0 {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid token type or amount"})
	}

	// Логирование входных данных
	log.Printf("Received request to subtract tokens from bot balance: token_type=%s, amount=%f",
		request.TokenType, request.Amount)

	// Вызов сервиса для уменьшения токенов из баланса бота
	err := c.GameService.SubtractTokensFromBotBalance(ctx.Request().Context(), request.TokenType, request.Amount)
	if err != nil {
		if err.Error() == "invalid token type" || err.Error() == "insufficient bot balance or no record found" {
			return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		log.Printf("Ошибка при вычитании токенов из баланса бота: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to subtract tokens from bot balance"})
	}

	// Возвращаем успешный ответ
	return ctx.JSON(http.StatusOK, map[string]string{"message": "Tokens subtracted from bot balance successfully"})
}
