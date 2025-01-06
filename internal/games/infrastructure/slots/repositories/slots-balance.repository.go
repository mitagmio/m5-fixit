package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Peranum/tg-dice/internal/games/infrastructure/slots/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SlotsBalanceRepository - Репозиторий для работы с балансом пользователя в тоннах и кубах.
type SlotsBalanceRepository struct {
	collection *mongo.Collection
}

// NewSlotsBalanceRepository - Конструктор репозитория для работы с коллекцией MongoDB.
func NewSlotsBalanceRepository(db *mongo.Database) *SlotsBalanceRepository {
	return &SlotsBalanceRepository{
		collection: db.Collection("slots_balance"), // Название коллекции для хранения баланса
	}
}

// InitializeBalance - Инициализация общего баланса.
func (repo *SlotsBalanceRepository) InitializeBalance(ctx context.Context, tons, cubes float64) error {
	// Создаем новый документ с балансом
	balance := entities.SlotsBalance{
		Tons:      tons,
		Cubes:     cubes,
		UpdatedAt: time.Now(),
	}

	// Обновляем или вставляем документ
	filter := bson.M{} // Нет фильтра по wallet
	update := bson.M{
		"$set": balance,
	}
	opts := options.Update().SetUpsert(true) // Вставить, если документа нет
	_, err := repo.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetBalance - Получить общий баланс.
func (repo *SlotsBalanceRepository) GetBalance(ctx context.Context) (*entities.SlotsBalance, error) {
	var balance entities.SlotsBalance
	filter := bson.M{} // Нет фильтра по wallet
	err := repo.collection.FindOne(ctx, filter).Decode(&balance)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Если документа нет, возвращаем nil
		}
		return nil, err // Если произошла ошибка, возвращаем ее
	}
	return &balance, nil
}

func (repo *SlotsBalanceRepository) UpdateBalance(ctx context.Context, tonsDelta, cubesDelta float64) error {
	// Обновляем только указанные поля (tons и cubes)
	update := bson.M{
		"$inc": bson.M{
			"tons":  tonsDelta,
			"cubes": cubesDelta,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}
	_, err := repo.collection.UpdateOne(ctx, bson.M{}, update) // Нет фильтра по wallet
	return err
}

func (repo *SlotsBalanceRepository) DeductTons(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return errors.New("сумма для вычитания должна быть положительной")
	}

	filter := bson.M{"tons": bson.M{"$gte": amount}}
	update := bson.M{
		"$inc": bson.M{"tons": -amount},
		"$set": bson.M{"updated_at": time.Now()},
	}

	result, err := repo.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("не удалось вычесть тонны: %v", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("недостаточно тонн на балансе")
	}

	return nil
}

// SubtractTokens вычитает указанное количество токенов указанного типа из баланса.
func (repo *SlotsBalanceRepository) SubtractTokens(ctx context.Context, tokenType string, amount float64) error {
	if amount <= 0 {
		return errors.New("сумма для вычитания должна быть положительной")
	}

	validTokens := map[string]bool{
		"tons":  true,
		"cubes": true,
	}

	if !validTokens[tokenType] {
		return errors.New("неверный тип токена")
	}

	filter := bson.M{tokenType: bson.M{"$gte": amount}}
	update := bson.M{
		"$inc": bson.M{tokenType: -amount},
		"$set": bson.M{"updated_at": time.Now()},
	}

	result, err := repo.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("не удалось вычесть токены %s: %v", tokenType, err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("недостаточно %s на балансе", tokenType)
	}

	return nil
}

// AddTokens добавляет указанное количество токенов указанного типа к балансу.
func (repo *SlotsBalanceRepository) AddTokens(ctx context.Context, tokenType string, amount float64) error {
	if amount <= 0 {
		return errors.New("сумма для добавления должна быть положительной")
	}

	validTokens := map[string]bool{
		"tons":  true,
		"cubes": true,
	}

	if !validTokens[tokenType] {
		return errors.New("неверный тип токена")
	}

	update := bson.M{
		"$inc": bson.M{tokenType: amount},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := repo.collection.UpdateOne(ctx, bson.M{}, update)
	if err != nil {
		return fmt.Errorf("не удалось добавить токены %s: %v", tokenType, err)
	}

	return nil
}
