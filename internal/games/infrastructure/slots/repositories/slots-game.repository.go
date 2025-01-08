package repositories

import (
	"context"
	"log"
	"time"

	"github.com/Peranum/tg-dice/internal/games/infrastructure/slots/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SlotGameRepository - Репозиторий для работы с игрой.
type SlotGameRepository struct {
	collection *mongo.Collection
}

// NewSlotGameRepository - Конструктор репозитория для SlotGame.
func NewSlotGameRepository(db *mongo.Database) *SlotGameRepository {
	return &SlotGameRepository{
		collection: db.Collection("slot_games"),
	}
}

// RecordGame - Записать информацию о сыгранной игре в MongoDB.
func (repo *SlotGameRepository) RecordGame(ctx context.Context, wallet string, bet float64, betType string, result string, winAmount float64) error {
	game := entities.SlotGame{
		Wallet:    wallet,
		Bet:       bet,
		BetType:   betType,
		Result:    result,
		WinAmount: winAmount,
		PlayedAt:  time.Now(),
	}

	res, err := repo.collection.InsertOne(ctx, game)
	log.Printf("Результат записи игры в БД: %v", res)
	if err != nil {
		log.Printf("Ошибка при сохранении записи игры в БД: %v", err)
		return err
	}

	return nil
}

// GetGamesByWallet - Получить все игры игрока по кошельку.
func (repo *SlotGameRepository) GetGamesByWallet(ctx context.Context, wallet string, limit int64) ([]entities.SlotGame, error) {
	var games []entities.SlotGame

	filter := bson.M{"wallet": wallet}
	options := options.Find().SetLimit(limit)

	cursor, err := repo.collection.Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var game entities.SlotGame
		if err := cursor.Decode(&game); err != nil {
			return nil, err
		}
		games = append(games, game)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return games, nil
}

// GetRecentGames - Получить последние игры по кошельку.
func (repo *SlotGameRepository) GetRecentGames(ctx context.Context, wallet string, limit int64) ([]entities.SlotGame, error) {
	var games []entities.SlotGame

	filter := bson.M{"wallet": wallet}
	options := options.Find().SetLimit(limit).SetSort(bson.M{"played_at": -1}) // Сортировка по времени (от новых к старым)

	cursor, err := repo.collection.Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var game entities.SlotGame
		if err := cursor.Decode(&game); err != nil {
			return nil, err
		}
		games = append(games, game)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return games, nil
}
