package services

import (
	"context"
	"log"
	"time"

	"github.com/Peranum/tg-dice/internal/games/infrastructure/history/entity"
	"github.com/Peranum/tg-dice/internal/games/infrastructure/history/repositories"
	"github.com/Peranum/tg-dice/internal/games/presentation/websockets/history" // WebSocket сервер
)

type GameService struct {
	gameRepo        *repositories.GameRepository
	websocketServer *history.WebSocketServer // WebSocket сервер для отправки обновлений
}

// NewGameService создает новый экземпляр GameService
func NewGameService(gameRepo *repositories.GameRepository, websocketServer *history.WebSocketServer) *GameService {
	return &GameService{
		gameRepo:        gameRepo,
		websocketServer: websocketServer,
	}
}

func (s *GameService) SaveGame(
	ctx context.Context,
	player1Name, player2Name string,
	player1Score, player2Score int,
	winner string,
	player1Earnings, player2Earnings float64,
	tokenType string,
	betAmount float64,
	player1Wallet, player2Wallet string, // Новые параметры: кошельки игроков
) error {
	// Создаём запись об игре
	gameRecord := &entities.GameRecord{
		Player1Name:     player1Name,
		Player2Name:     player2Name,
		Player1Score:    player1Score,
		Player2Score:    player2Score,
		Winner:          winner,
		Player1Earnings: player1Earnings,
		Player2Earnings: player2Earnings,
		TokenType:       tokenType,
		BetAmount:       betAmount,
		Player1Wallet:   player1Wallet, // Устанавливаем кошелёк игрока 1
		Player2Wallet:   player2Wallet, // Устанавливаем кошелёк игрока 2
		TimePlayed:      time.Now(),
	}

	// Сохраняем запись игры через репозиторий
	err := s.gameRepo.Save(ctx, gameRecord)
	if err != nil {
		log.Printf("Ошибка при сохранении игры: %v", err)
		return err
	}

	// Подготавливаем информацию для WebSocket
	gameInfo := map[string]interface{}{
		"Player1Name":     player1Name,
		"Player2Name":     player2Name,
		"Player1Score":    player1Score,
		"Player2Score":    player2Score,
		"Winner":          winner,
		"Player1Earnings": player1Earnings,
		"Player2Earnings": player2Earnings,
		"TokenType":       tokenType,
		"BetAmount":       betAmount,
		"Player1Wallet":   player1Wallet, // Передаём кошелёк игрока 1
		"Player2Wallet":   player2Wallet, // Передаём кошелёк игрока 2
		"TimePlayed":      gameRecord.TimePlayed.Format(time.RFC3339),
		"Counter":         gameRecord.Counter, // Добавляем поле counter
	}

	// Отправляем информацию об игре всем подключённым WebSocket клиентам
	s.websocketServer.Broadcast(gameInfo)

	return nil
}

// GetGamesHistory получает общую историю всех игр
func (s *GameService) GetGamesHistory(ctx context.Context, limit int) ([]*entities.GameRecord, error) {
	// Получаем общую историю игр из репозитория
	games, err := s.gameRepo.GetAllGamesHistory(ctx, limit)
	if err != nil {
		log.Printf("Ошибка при получении общей истории игр: %v", err)
		return nil, err
	}
	return games, nil
}

// GetUserGameHistory получает историю игр для конкретного пользователя по кошельку
func (s *GameService) GetUserGameHistory(ctx context.Context, wallet string, limit int) ([]*entities.GameRecord, error) {
	// Получаем историю игр для конкретного пользователя
	games, err := s.gameRepo.GetGameHistoryByWallet(ctx, wallet, limit)
	if err != nil {
		log.Printf("Ошибка при получении истории игр для кошелька %s: %v", wallet, err)
		return nil, err
	}
	return games, nil
}
