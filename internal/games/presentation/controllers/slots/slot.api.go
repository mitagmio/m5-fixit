package slots

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Peranum/tg-dice/internal/games/domain/slots/services"
	"github.com/labstack/echo/v4"
)

// SlotGameController - Контроллер для работы с играми слотов.
type SlotGameController struct {
	SlotGameService     *services.SlotGameService
	SlotsBalanceService *services.SlotsBalanceService
}

// NewSlotGameController - Конструктор для создания нового SlotGameController.
func NewSlotGameController(slotGameService *services.SlotGameService, slotsBalanceService *services.SlotsBalanceService) *SlotGameController {
	return &SlotGameController{
		SlotGameService:     slotGameService,
		SlotsBalanceService: slotsBalanceService,
	}
}

// InitializeBalanceRequest - структура для данных запроса на инициализацию баланса.
type InitializeBalanceRequest struct {
	Tons  float64 `json:"tons"`  // Баланс в тоннах
	Cubes float64 `json:"cubes"` // Баланс в кубах
}

// PlaySlotRequest - структура для данных запроса игры в слоты.
type PlaySlotRequest struct {
	Wallet string  `json:"wallet" validate:"required"` // Кошелек пользователя
	Ton    float64 `json:"ton,omitempty"`              // Ставка в тоннах (опционально)
	Cubes  int     `json:"cubes,omitempty"`            // Ставка в кубах (опционально)
}

// PlaySlot - Контроллер для игры в слоты.
// @Summary Запуск игры в слоты
// @Description Выполнение игры в слоты с выбором ставки
// @Tags Slots
// @Accept json
// @Produce json
// @Param playSlotRequest body PlaySlotRequest true "Параметры игры"
// @Success 200 {object} PlaySlotResponse "Результат игры"
// @Failure 400 {object} ErrorResponse "Ошибка с некорректной ставкой"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /slots/play [post]
func (controller *SlotGameController) PlaySlot(c echo.Context) error {
	var request PlaySlotRequest

	// Привязываем данные из тела запроса
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request data"})
	}

	// Проверяем, что передан только один из параметров Ton или Cubes
	if (request.Ton > 0 && request.Cubes > 0) || (request.Ton == 0 && request.Cubes == 0) {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Specify either Ton or Cubes, but not both"})
	}

	// Вызов сервиса для игры в слоты
	resultCombo, winAmount, err := controller.SlotGameService.PlaySlot(c.Request().Context(), request.Wallet, request.Ton, request.Cubes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("Failed to play slot: %v", err)})
	}

	// Возвращаем результат
	return c.JSON(http.StatusOK, PlaySlotResponse{
		ResultCombo: resultCombo,
		WinAmount:   winAmount,
	})
}

type RecordGameRequest struct {
	Wallet    string  `json:"wallet"`     // Кошелек игрока
	Bet       float64 `json:"bet"`        // Ставка игрока
	Result    string  `json:"result"`     // Результат игры
	WinAmount float64 `json:"win_amount"` // Выигрыш игрока
}

// GetGamesByWallet - Контроллер для получения всех игр по кошельку.
// @Summary Получить все игры по кошельку
// @Description Получить все игры, сыгранные пользователем по его кошельку
// @Tags Slots
// @Accept json
// @Produce json
// @Param wallet path string true "Кошелек игрока"
// @Param limit query int true "Лимит количества игр"
// @Success 200 {array} GameRecord "Список игр"
// @Failure 400 {object} ErrorResponse "Ошибка с некорректным лимитом"
// @Failure 500 {object} ErrorResponse "Ошибка сервера при получении игр"
// @Router /slots/{wallet}/games [get]
func (controller *SlotGameController) GetGamesByWallet(c echo.Context) error {
	wallet := c.Param("wallet")
	limit, err := strconv.ParseInt(c.QueryParam("limit"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid limit"})
	}

	// Вызов сервиса для получения игр по кошельку
	games, err := controller.SlotGameService.GetGamesByWallet(c.Request().Context(), wallet, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("failed to get games by wallet: %v", err)})
	}

	// Возвращаем список игр
	return c.JSON(http.StatusOK, games)
}

// GetRecentGames - Контроллер для получения последних игр по кошельку.
// @Summary Получить последние игры по кошельку
// @Description Получить последние игры пользователя по его кошельку
// @Tags Slots
// @Accept json
// @Produce json
// @Param wallet path string true "Кошелек игрока"
// @Param limit query int true "Лимит количества игр"
// @Success 200 {array} GameRecord "Список последних игр"
// @Failure 400 {object} ErrorResponse "Ошибка с некорректным лимитом"
// @Failure 500 {object} ErrorResponse "Ошибка сервера при получении последних игр"
// @Router /slots/{wallet}/recent-games [get]
func (controller *SlotGameController) GetRecentGames(c echo.Context) error {
	wallet := c.Param("wallet")
	limit, err := strconv.ParseInt(c.QueryParam("limit"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid limit"})
	}

	// Вызов сервиса для получения последних игр по кошельку
	games, err := controller.SlotGameService.GetRecentGames(c.Request().Context(), wallet, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("failed to get recent games: %v", err)})
	}

	// Возвращаем список последних игр
	return c.JSON(http.StatusOK, games)
}

// PlaySlotResponse - Ответ на запрос игры в слоты
// PlaySlotResponse - Ответ на запрос игры в слоты
type PlaySlotResponse struct {
	ResultCombo []int   `json:"result_combo"` // Комбинация чисел
	WinAmount   float64 `json:"win_amount"`   // Выигрыш
}

// ErrorResponse - Структура ошибки для возврата пользователю
type ErrorResponse struct {
	Message string `json:"message"` // Сообщение об ошибке
}

// SuccessResponse - Структура для успешного ответа
type SuccessResponse struct {
	Status string `json:"status"` // Статус успешного выполнения
}

// GameRecord - Структура для игры
type GameRecord struct {
	Wallet    string  `json:"wallet"`     // Кошелек игрока
	FirstName string  `bson:"first_name"` // Имя игрока
	Bet       float64 `json:"bet"`        // Ставка
	BetType   string  `bson:"bet_type"`   // Тип ставки (ton или cubes)
	Result    string  `json:"result"`     // Результат игры
	WinAmount float64 `json:"win_amount"` // Выигрыш
	Timestamp string  `json:"timestamp"`  // Время игры
}

// InitializeBalance - Контроллер для инициализации общего баланса.
// @Summary Инициализация баланса
// @Description Устанавливает общий баланс в тоннах и кубах
// @Tags Slots
// @Accept json
// @Produce json
// @Param initializeBalanceRequest body InitializeBalanceRequest true "Данные для инициализации баланса"
// @Success 200 {object} SuccessResponse "Баланс успешно инициализирован"
// @Failure 400 {object} ErrorResponse "Некорректные данные запроса"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /slots/balance/initialize [post]
func (controller *SlotGameController) InitializeBalance(c echo.Context) error {
	var request InitializeBalanceRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request data"})
	}

	err := controller.SlotsBalanceService.InitializeBalance(c.Request().Context(), request.Tons, request.Cubes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("Failed to initialize balance: %v", err)})
	}

	return c.JSON(http.StatusOK, SuccessResponse{Status: "Balance successfully initialized"})
}

// GetBalance - Контроллер для получения общего баланса.
// @Summary Получение общего баланса
// @Description Возвращает общий баланс в тоннах и кубах
// @Tags Slots
// @Accept json
// @Produce json
// @Success 200 {object} SlotsBalanceResponse "Общий баланс"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /slots/balance [get]
func (controller *SlotGameController) GetBalance(c echo.Context) error {
	balance, err := controller.SlotsBalanceService.GetBalance(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("Failed to get balance: %v", err)})
	}

	return c.JSON(http.StatusOK, SlotsBalanceResponse{
		Tons:      balance.Tons,
		Cubes:     balance.Cubes,
		UpdatedAt: balance.UpdatedAt,
	})
}

// UpdateBalanceRequest - структура для данных запроса на обновление баланса.
type UpdateBalanceRequest struct {
	TonsDelta  float64 `json:"tons_delta"`  // Изменение баланса в тоннах
	CubesDelta float64 `json:"cubes_delta"` // Изменение баланса в кубах
}

// UpdateBalance - Контроллер для обновления общего баланса.
// @Summary Обновление баланса
// @Description Изменяет общий баланс в тоннах и кубах
// @Tags Slots
// @Accept json
// @Produce json
// @Param updateBalanceRequest body UpdateBalanceRequest true "Данные для обновления баланса"
// @Success 200 {object} SuccessResponse "Баланс успешно обновлен"
// @Failure 400 {object} ErrorResponse "Некорректные данные запроса"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /slots/balance/update [patch]
func (controller *SlotGameController) UpdateBalance(c echo.Context) error {
	var request UpdateBalanceRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request data"})
	}

	err := controller.SlotsBalanceService.UpdateBalance(c.Request().Context(), request.TonsDelta, request.CubesDelta)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("Failed to update balance: %v", err)})
	}

	return c.JSON(http.StatusOK, SuccessResponse{Status: "Balance successfully updated"})
}

// SlotsBalanceResponse - Ответ на запрос баланса
type SlotsBalanceResponse struct {
	Tons      float64   `json:"tons"`       // Баланс в тоннах
	Cubes     float64   `json:"cubes"`      // Баланс в кубах
	UpdatedAt time.Time `json:"updated_at"` // Время последнего обновления
}

// TokenOperationRequest - структура для данных запроса на операции с токенами.
type TokenOperationRequest struct {
	Amount    float64 `json:"amount" validate:"required,gt=0"` // Сумма токенов
	TokenType string  `json:"token_type" validate:"required"`  // Тип токенов (tons или cubes)
}

// SubtractTokens - Контроллер для вычитания токенов.
// @Summary Вычитание токенов
// @Description Вычитает указанное количество токенов из баланса
// @Tags Slots
// @Accept json
// @Produce json
// @Param tokenOperationRequest body TokenOperationRequest true "Данные для вычитания токенов"
// @Success 200 {object} SuccessResponse "Токены успешно вычтены"
// @Failure 400 {object} ErrorResponse "Некорректные данные запроса"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /slots/balance/subtract [post]
func (controller *SlotGameController) SubtractTokens(c echo.Context) error {
	var request TokenOperationRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request data"})
	}

	if request.TokenType != "tons" && request.TokenType != "cubes" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid token type. Must be 'tons' or 'cubes'"})
	}

	err := controller.SlotsBalanceService.SubtractTokens(c.Request().Context(), request.TokenType, request.Amount)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("Failed to subtract tokens: %v", err)})
	}

	return c.JSON(http.StatusOK, SuccessResponse{Status: "Tokens successfully subtracted"})
}

// AddTokens - Контроллер для добавления токенов.
// @Summary Добавление токенов
// @Description Добавляет указанное количество токенов к балансу
// @Tags Slots
// @Accept json
// @Produce json
// @Param tokenOperationRequest body TokenOperationRequest true "Данные для добавления токенов"
// @Success 200 {object} SuccessResponse "Токены успешно добавлены"
// @Failure 400 {object} ErrorResponse "Некорректные данные запроса"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /slots/balance/add [post]
func (controller *SlotGameController) AddTokens(c echo.Context) error {
	var request TokenOperationRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request data"})
	}

	if request.TokenType != "tons" && request.TokenType != "cubes" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid token type. Must be 'tons' or 'cubes'"})
	}

	err := controller.SlotsBalanceService.AddTokens(c.Request().Context(), request.TokenType, request.Amount)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("Failed to add tokens: %v", err)})
	}

	return c.JSON(http.StatusOK, SuccessResponse{Status: "Tokens successfully added"})
}
