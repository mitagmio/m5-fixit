package repositories

import (
	"context"
	"errors"
	"github.com/Peranum/tg-dice/internal/games/infrastructure/bot/entity"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// BotRepository представляет репозиторий для управления балансом бота
type BotRepository struct {
	Collection *mongo.Collection
}

func NewBotRepository(db *mongo.Database) *BotRepository {
	return &BotRepository{
		Collection: db.Collection("bot_balances"),
	}
}

// GetTokenBalance получает баланс конкретного токена
// GetTokenBalance получает баланс конкретного токена
func (br *BotRepository) GetTokenBalance(ctx context.Context, tokenType string) (float64, error) {
	// Валидация типа токена
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		return 0, errors.New("invalid token type")
	}

	// Поиск записи с балансами
	var result entities.BotBalanceEntity

	err := br.Collection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, errors.New("no bot balance found")
		}
		return 0, err
	}

	// Возвращаем баланс указанного токена
	switch tokenType {
	case "ton_balance":
		return result.TonBalance, nil
	case "m5_balance":
		return result.M5Balance, nil
	case "dfc_balance":
		return result.DfcBalance, nil
	default:
		return 0, errors.New("invalid token type")
	}
}

func (br *BotRepository) AddTokenBalance(ctx context.Context, tokenType string, amount float64) error {
	// Валидация типа токена
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		return errors.New("invalid token type")
	}

	// Увеличение значения указанного токена
	update := bson.M{
		"$inc": bson.M{
			tokenType: amount,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// Обновление первой записи (обычно запись с балансом бота одна)
	result, err := br.Collection.UpdateOne(ctx, bson.M{}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("no bot balance found to update")
	}

	return nil
}

func (br *BotRepository) CreateBotBalance(ctx context.Context, tonBalance, m5Balance, dfcBalance float64) error {
	// Создаем объект для записи
	newBotBalance := bson.M{
		"ton_balance": tonBalance,
		"m5_balance":  m5Balance,
		"dfc_balance": dfcBalance,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}

	// Вставляем объект в коллекцию
	_, err := br.Collection.InsertOne(ctx, newBotBalance)
	if err != nil {
		return err
	}

	return nil
}

// GetBotBalance получает все балансы бота
func (br *BotRepository) GetBotBalance(ctx context.Context) (entities.BotBalanceEntity, error) {
	var result entities.BotBalanceEntity

	err := br.Collection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entities.BotBalanceEntity{}, errors.New("no bot balance found")
		}
		return entities.BotBalanceEntity{}, err
	}

	return result, nil
}

func (br *BotRepository) SubtractTokenBalance(ctx context.Context, tokenType string, amount float64) error {
	// Валидация типа токена
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		return errors.New("invalid token type")
	}

	// Уменьшение значения указанного токена
	update := bson.M{
		"$inc": bson.M{
			tokenType: -amount,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// Условие для проверки, что баланс не уйдет в минус
	filter := bson.M{
		tokenType: bson.M{"$gte": amount},
	}

	// Обновление первой записи
	result, err := br.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("insufficient bot balance or no record found")
	}

	return nil
}
