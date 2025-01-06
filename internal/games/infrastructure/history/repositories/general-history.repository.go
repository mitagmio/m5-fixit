package repositories

import (
	"context"
	"github.com/Peranum/tg-dice/internal/games/infrastructure/history/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

type GameRepository struct {
	collection        *mongo.Collection
	counterCollection *mongo.Collection // Коллекция для отслеживания инкремента
}

// NewGameRepository создает новый экземпляр репозитория для игры
const initialCounterValue = 14000

// NewGameRepository создает новый экземпляр репозитория для игры
func NewGameRepository(db *mongo.Database) *GameRepository {
	return &GameRepository{
		collection:        db.Collection("game_history"),
		counterCollection: db.Collection("game_counter"), // Коллекция для счетчиков
	}
}

func (r *GameRepository) getNextGameNumber(ctx context.Context) (int, error) {
	// Step 1: Ensure the counter document exists
	filter := bson.M{"_id": "gameCounter"}
	initUpdate := bson.M{
		"$setOnInsert": bson.M{"_id": "gameCounter", "counter": 13999}, // Initialize counter if not present
	}

	initOpts := options.Update().SetUpsert(true)

	_, err := r.counterCollection.UpdateOne(ctx, filter, initUpdate, initOpts)
	if err != nil {
		log.Printf("Ошибка при инициализации счетчика: %v", err)
		return 0, err
	}

	// Step 2: Increment the counter and return the new value
	incrementUpdate := bson.M{
		"$inc": bson.M{"counter": 1}, // Increment counter by 1
	}

	findOpts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var result struct {
		Counter int `bson:"counter"`
	}

	err = r.counterCollection.FindOneAndUpdate(ctx, filter, incrementUpdate, findOpts).Decode(&result)
	if err != nil {
		log.Printf("Ошибка при увеличении счетчика: %v", err)
		return 0, err
	}

	log.Printf("Следующий номер игры: %d", result.Counter)
	return result.Counter, nil
}

// Save сохраняет запись игры в базе данных и инкрементирует счетчик
func (r *GameRepository) Save(ctx context.Context, game *entities.GameRecord) error {
	// Получаем следующий номер игры (с инкрементом)
	gameNumber, err := r.getNextGameNumber(ctx)
	if err != nil {
		log.Printf("Ошибка при получении следующего номера игры: %v", err)
		return err
	}

	// Присваиваем номер игры
	game.Counter = gameNumber

	// Сохраняем запись игры
	_, err = r.collection.InsertOne(ctx, game)
	if err != nil {
		log.Printf("Ошибка при сохранении записи игры в БД: %v", err)
		return err
	}
	return nil
}

// Получение всех игр (для примера)
func (r *GameRepository) GetAllGamesHistory(ctx context.Context, limit int) ([]*entities.GameRecord, error) {
	var games []*entities.GameRecord

	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{
		{Key: "time_played", Value: -1}, // Сортировка по времени (от новых к старым)
	})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		log.Printf("Ошибка при получении общей истории игр из БД: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var game entities.GameRecord
		if err := cursor.Decode(&game); err != nil {
			log.Printf("Ошибка при декодировании записи игры: %v", err)
			continue
		}
		games = append(games, &game)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Ошибка при чтении курсора: %v", err)
		return nil, err
	}

	return games, nil
}

// GetGameHistoryByWallet получает историю игр для пользователя по его кошельку
func (r *GameRepository) GetGameHistoryByWallet(ctx context.Context, wallet string, limit int) ([]*entities.GameRecord, error) {
	var games []*entities.GameRecord

	// Фильтр для поиска по кошельку игрока
	filter := bson.M{
		"$or": []bson.M{
			{"player1_wallet": wallet},
			{"player2_wallet": wallet},
		},
	}

	// Опции для сортировки и ограничения количества записей
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{
		{Key: "time_played", Value: -1}, // Сортируем по времени (от новых к старым)
	})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("Ошибка при получении истории игр из БД: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var game entities.GameRecord
		if err := cursor.Decode(&game); err != nil {
			log.Printf("Ошибка при декодировании записи игры: %v", err)
			continue
		}
		games = append(games, &game)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Ошибка при чтении курсора: %v", err)
		return nil, err
	}

	return games, nil
}

// GetGameHistoryByTokenType получает историю игр по определённому типу токена
func (r *GameRepository) GetGameHistoryByTokenType(ctx context.Context, tokenType string, limit int) ([]*entities.GameRecord, error) {
	var games []*entities.GameRecord

	// Фильтр для поиска по типу токена
	filter := bson.M{"token_type": tokenType}

	// Опции для сортировки и ограничения количества записей
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{
		{Key: "time_played", Value: -1}, // Сортируем по времени (от новых к старым)
	})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("Ошибка при получении истории игр по типу токена: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var game entities.GameRecord
		if err := cursor.Decode(&game); err != nil {
			log.Printf("Ошибка при декодировании записи игры: %v", err)
			continue
		}
		games = append(games, &game)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Ошибка при чтении курсора: %v", err)
		return nil, err
	}

	return games, nil
}
