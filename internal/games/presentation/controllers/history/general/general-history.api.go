package general

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Peranum/tg-dice/internal/games/domain/history/services"
	"github.com/labstack/echo/v4"
)

// GameRecord структура для данных об игре
type GameRecord struct {
	Player1Name     string  `json:"player1_name"`     // Имя первого игрока
	Player2Name     string  `json:"player2_name"`     // Имя второго игрока
	Player1Score    int     `json:"player1_score"`    // Счет первого игрока
	Player2Score    int     `json:"player2_score"`    // Счет второго игрока
	Winner          string  `json:"winner"`           // Победитель
	Player1Earnings float64 `json:"player1_earnings"` // Заработок первого игрока
	Player2Earnings float64 `json:"player2_earnings"` // Заработок второго игрока
	TimePlayed      string  `json:"time_played"`      // Время игры (формат времени: RFC3339)
	TokenType       string  `json:"token_type"`       // Тип токена
	BetAmount       float64 `json:"bet_amount"`       // Сумма ставки
	Player1Wallet   string  `json:"player1_wallet"`   // Кошелек первого игрока
	Player2Wallet   string  `json:"player2_wallet"`   // Кошелек второго игрока
	Counter         int     `json:"counter"`          // Changed to int64

}

// GameHistoryResponse - структура для ответа API
type GameHistoryResponse struct {
	Player1Name     string    `json:"player1_name"`     // изменено с Player1Name
	Player2Name     string    `json:"player2_name"`     // изменено с Player2Name
	Player1Score    int       `json:"player1_score"`    // изменено с Player1Score
	Player2Score    int       `json:"player2_score"`    // изменено с Player2Score
	Winner          string    `json:"winner"`           // изменено с Winner
	Player1Earnings float64   `json:"player1_earnings"` // изменено с Player1Earnings
	Player2Earnings float64   `json:"player2_earnings"` // изменено с Player2Earnings
	TimePlayed      time.Time `json:"time_played"`      // изменено с TimePlayed
	TokenType       string    `json:"token_type"`       // изменено с TokenType
	BetAmount       float64   `json:"bet_amount"`       // изменено с BetAmount
	Counter         int       `json:"counter"`          // изменено с Counter
}

type GameHistoryController struct {
	gameService *services.GameService
}

// NewGameHistoryController создает новый контроллер для истории игр
func NewGameHistoryController(gameService *services.GameService) *GameHistoryController {
	return &GameHistoryController{
		gameService: gameService,
	}
}

// SaveGame создает запись об игре
// @Summary Создает запись об игре
// @Description Сохраняет данные игры (имена игроков, счета, победителя, заработанные средства, тип токена, сумму ставки)
// @Tags game-history
// @Accept  json
// @Produce  json
// @Param gameRecord body GameRecord true "Информация об игре"
// @Success 200 {string} string "Игра сохранена успешно"
// @Failure 400 {string} string "Некорректные данные"
// @Failure 500 {string} string "Ошибка при сохранении игры"
// @Router /games/history [post]
func (c *GameHistoryController) SaveGame(ctx echo.Context) error {
	var gameRecord GameRecord

	// Привязываем данные из тела запроса к структуре gameRecord
	if err := ctx.Bind(&gameRecord); err != nil {
		log.Printf("Ошибка при привязке данных: %v", err)
		return ctx.JSON(http.StatusBadRequest, "Некорректные данные")
	}

	// Сохраняем игру
	err := c.gameService.SaveGame(
		ctx.Request().Context(),
		gameRecord.Player1Name,
		gameRecord.Player2Name,
		gameRecord.Player1Score,
		gameRecord.Player2Score,
		gameRecord.Winner,
		gameRecord.Player1Earnings,
		gameRecord.Player2Earnings,
		gameRecord.TokenType,
		gameRecord.BetAmount,
		gameRecord.Player1Wallet, // Передаем кошелек первого игрока
		gameRecord.Player2Wallet, // Передаем кошелек второго игрока
	)
	if err != nil {
		log.Printf("Ошибка при сохранении игры: %v", err)
		return ctx.JSON(http.StatusInternalServerError, "Ошибка при сохранении игры")
	}

	return ctx.JSON(http.StatusOK, "Игра сохранена успешно")
}

// GetGamesHistory получает общую историю всех игр
// @Summary Получить историю всех игр
// @Description Получить список всех игр с ограничением по количеству
// @Tags History
// @Accept json
// @Produce json
// @Param limit query int false "Лимит количества игр" default(50)
// @Success 200 {array} GameHistoryResponse
// @Failure 500 {string} string "Ошибка при получении истории игр"
// @Router /games/history [get]
func (c *GameHistoryController) GetGamesHistory(ctx echo.Context) error {
	// Получаем параметр limit из запроса
	limitParam := ctx.QueryParam("limit")
	limit := 50 // Значение по умолчанию

	if limitParam != "" {
		parsedLimit, err := strconv.Atoi(limitParam)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	games, err := c.gameService.GetGamesHistory(ctx.Request().Context(), limit)
	if err != nil {
		log.Printf("Ошибка при получении общей истории игр: %v", err)
		return ctx.JSON(http.StatusInternalServerError, "Ошибка при получении истории игр")
	}

	// Преобразуем данные в формат ответа
	var response []GameHistoryResponse
	for _, game := range games {
		historyItem := GameHistoryResponse{
			Player1Name:     game.Player1Name,
			Player2Name:     game.Player2Name,
			Player1Score:    game.Player1Score,
			Player2Score:    game.Player2Score,
			Winner:          game.Winner,
			Player1Earnings: game.Player1Earnings,
			Player2Earnings: game.Player2Earnings,
			TimePlayed:      game.TimePlayed,
			TokenType:       game.TokenType,
			BetAmount:       game.BetAmount,
			Counter:         game.Counter,
		}
		response = append(response, historyItem)
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetUserGameHistory получает историю игр для конкретного пользователя по кошельку
// @Summary Получает историю игр пользователя
// @Description Возвращает список последних игр для конкретного пользователя по кошельку с ограничением по количеству
// @Tags game-history
// @Accept  json
// @Produce  json
// @Param wallet path string true "Кошелек пользователя"
// @Param limit query int false "Лимит количества записей" default(50)
// @Success 200 {array} GameRecord
// @Failure 400 {string} string "Некорректный кошелек"
// @Failure 500 {string} string "Ошибка при получении истории игр"
// @Router /games/history/{wallet} [get]
func (c *GameHistoryController) GetUserGameHistory(ctx echo.Context) error {
	wallet := ctx.Param("wallet")
	if wallet == "" {
		return ctx.JSON(http.StatusBadRequest, "Кошелек обязателен")
	}

	// Получаем параметр limit из запроса
	limitParam := ctx.QueryParam("limit")
	limit := 50 // Значение по умолчанию

	if limitParam != "" {
		parsedLimit, err := strconv.Atoi(limitParam)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	games, err := c.gameService.GetUserGameHistory(ctx.Request().Context(), wallet, limit)
	if err != nil {
		log.Printf("Ошибка при получении истории игр для кошелька %s: %v", wallet, err)
		return ctx.JSON(http.StatusInternalServerError, "Ошибка при получении истории игр")
	}

	return ctx.JSON(http.StatusOK, games)
}
