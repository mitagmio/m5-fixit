package repositories

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Peranum/tg-dice/internal/user/infrastructure/odm-entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	Collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		Collection: db.Collection("users"),
	}
}

func (ur *UserRepository) Create(ctx context.Context, user *odm_entities.UserEntity) (*odm_entities.UserEntity, error) {
	log.Printf("[CreateUser] Checking TgID uniqueness: tgID=%s", user.TgID)

	// Проверяем уникальность TgID
	existingUser := odm_entities.UserEntity{}
	err := ur.Collection.FindOne(ctx, bson.M{"tgid": user.TgID}).Decode(&existingUser)
	if err == nil {
		log.Printf("[CreateUser] TgID already exists: %s", user.TgID)
		return nil, errors.New("user with this TgID already exists")
	}
	if err != mongo.ErrNoDocuments && err != nil {
		log.Printf("[CreateUser] Error checking TgID: %v", err)
		return nil, fmt.Errorf("error checking for existing TgID: %v", err)
	}

	// Инициализация ReferralEarnings
	user.ReferralEarnings = map[string]float64{
		"ton_balance": 0.0,
		"m5_balance":  0.0,
		"dfc_balance": 0.0,
	}

	// Устанавливаем временные метки
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// Вставляем пользователя
	log.Printf("[CreateUser] Inserting user into database: %+v", user)
	result, err := ur.Collection.InsertOne(ctx, user)
	if err != nil {
		log.Printf("[CreateUser] Insert error: %v", err)
		return nil, err
	}

	// Устанавливаем сгенерированный ObjectID
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		user.ID = oid
	} else {
		log.Printf("[CreateUser] Failed to retrieve ObjectID")
		return nil, errors.New("failed to retrieve ObjectID from result")
	}

	log.Printf("[CreateUser] User inserted successfully: %+v", user)
	return user, nil
}

func (ur *UserRepository) GetTokenBalance(ctx context.Context, wallet string, tokenType string) (float64, error) {
	// Валидация типа токена
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		return 0, errors.New("invalid token type")
	}

	// Ищем пользователя по полю wallet
	var user odm_entities.UserEntity
	err := ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, errors.New("user not found")
		}
		return 0, err
	}

	// Возвращаем баланс указанного токена
	switch tokenType {
	case "ton_balance":
		return user.Ton_balance, nil
	case "m5_balance":
		return user.M5_balance, nil
	case "dfc_balance":
		return user.Dfc_balance, nil
	default:
		return 0, errors.New("unexpected token type")
	}
}

func (ur *UserRepository) GetByID(ctx context.Context, id string) (*odm_entities.UserEntity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var user odm_entities.UserEntity
	// Ищем пользователя по _id
	err = ur.Collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) UpdateByTgID(ctx context.Context, tgid string, updateData map[string]interface{}) error {
	// Обновляем время последнего обновления
	updateData["updated_at"] = time.Now()

	// Выполняем обновление пользователя по tgID
	filter := bson.M{"tgid": tgid}
	_, err := ur.Collection.UpdateOne(
		ctx,
		filter,
		bson.M{"$set": updateData},
	)
	return err
}

func (ur *UserRepository) Delete(ctx context.Context, id string) error {
	// Преобразуем строку в ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// Удаляем пользователя по _id
	_, err = ur.Collection.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

func (ur *UserRepository) List(ctx context.Context, limit int64, offset int64) ([]odm_entities.UserEntity, error) {
	// Настройка параметров для пагинации
	opts := options.Find()
	opts.SetLimit(limit)
	opts.SetSkip(offset)

	// Выполняем поиск всех пользователей с учетом пагинации
	cursor, err := ur.Collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []odm_entities.UserEntity
	// Проходим по курсору и добавляем пользователей в срез
	for cursor.Next(ctx) {
		var user odm_entities.UserEntity
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	// Проверяем на ошибку в процессе чтения курсора
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (ur *UserRepository) GetByWallet(ctx context.Context, wallet string) (*odm_entities.UserEntity, error) {
	var user odm_entities.UserEntity
	// Ищем пользователя по полю wallet
	err := ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) AddTokens(ctx context.Context, wallet string, tokenUpdates map[string]float64) error {
	// Логируем входные данные
	log.Printf("[AddTokens] Updating tokens for wallet: %s, updates: %+v", wallet, tokenUpdates)

	// Проверяем, что переданы только валидные токены
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	// Строим условие фильтрации, чтобы избежать отрицательных балансов
	filter := bson.M{"wallet": wallet}
	for token, amount := range tokenUpdates {
		if !validTokens[token] {
			log.Printf("[AddTokens] Invalid token type: %s", token)
			return errors.New("invalid token type")
		}
		if amount < 0 {
			// Текущий баланс должен быть >= -amount, чтобы избежать отрицательного баланса после обновления
			filter[token] = bson.M{"$gte": -amount}
		}
	}

	// Строим поле обновления
	updateFields := bson.M{}
	for token, amount := range tokenUpdates {
		updateFields[token] = amount
	}

	// Выполняем обновление с условиями фильтрации
	result, err := ur.Collection.UpdateOne(ctx, filter, bson.M{
		"$inc": updateFields,
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	})

	if err != nil {
		log.Printf("[AddTokens] MongoDB error: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		log.Printf("[AddTokens] Update failed due to insufficient balance or user not found for wallet: %s", wallet)
		return errors.New("insufficient balance or user not found")
	}

	log.Printf("[AddTokens] Tokens updated successfully for wallet: %s", wallet)
	return nil
}


func (ur *UserRepository) AddCubes(ctx context.Context, wallet string, cubes int) error {
	// Находим текущего пользователя по кошельку
	var user odm_entities.UserEntity
	err := ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}).Decode(&user)
	if err != nil {
		return errors.New("user not found")
	}

	// Проверяем, чтобы итоговый баланс кубов не стал отрицательным
	if user.Cubes+cubes < 0 {
		return errors.New("cubes cannot go negative")
	}

	// Увеличиваем количество кубов
	_, err = ur.Collection.UpdateOne(
		ctx,
		bson.M{"wallet": wallet},
		bson.M{
			"$inc": bson.M{
				"cubes": cubes,
			},
			"$set": bson.M{
				"updated_at": time.Now(),
			},
		},
	)

	return err
}

func (r *UserRepository) DoesUserExist(ctx context.Context, wallet string) (bool, error) {
	var count int64
	count, err := r.Collection.CountDocuments(ctx, bson.M{"wallet": wallet})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (ur *UserRepository) HasSufficientBalance(ctx context.Context, wallet string, tokenType string, amount float64) (bool, error) {
	// Валидация типа токена
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		return false, errors.New("invalid token type")
	}

	// Логируем входные параметры
	log.Printf("[HasSufficientBalance] Checking balance for wallet: %s, tokenType: %s, requiredAmount: %f", wallet, tokenType, amount)

	// Проверяем существование пользователя
	exists, err := ur.DoesUserExist(ctx, wallet)
	if err != nil {
		log.Printf("[HasSufficientBalance] Error checking user existence for wallet: %s, error: %v", wallet, err)
		return false, fmt.Errorf("error checking user existence: %v", err)
	}

	if !exists {
		log.Printf("[HasSufficientBalance] User not found for wallet: %s", wallet)
		return false, errors.New("user not found")
	}

	// Ищем пользователя по кошельку
	var user odm_entities.UserEntity
	err = ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}).Decode(&user)
	if err != nil {
		log.Printf("[HasSufficientBalance] Error fetching user: %v", err)
		return false, fmt.Errorf("error fetching user: %v", err)
	}

	// Логируем найденного пользователя
	log.Printf("[HasSufficientBalance] User found: %+v", user)

	// Проверяем, достаточно ли баланса
	var currentBalance float64
	switch tokenType {
	case "ton_balance":
		currentBalance = user.Ton_balance
	case "m5_balance":
		currentBalance = user.M5_balance
	case "dfc_balance":
		currentBalance = user.Dfc_balance
	}

	log.Printf("[HasSufficientBalance] Current balance for %s: %f, required: %f", tokenType, currentBalance, amount)

	if currentBalance >= amount {
		log.Printf("[HasSufficientBalance] Sufficient balance for wallet: %s", wallet)
		return true, nil
	}

	log.Printf("[HasSufficientBalance] Insufficient balance for wallet: %s", wallet)
	return false, nil
}

func (ur *UserRepository) GetUserBalances(ctx context.Context, wallet string) (map[string]interface{}, error) {
	var user odm_entities.UserEntity
	err := ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	// Возвращаем балансы токенов и кубов
	balances := map[string]interface{}{
		"ton_balance": user.Ton_balance,
		"m5_balance":  user.M5_balance,
		"dfc_balance": user.Dfc_balance,
		"cubes":       user.Cubes,
	}
	return balances, nil
}

func (repo *UserRepository) GetUsersByReferredBy(ctx context.Context, referredBy string) ([]*odm_entities.UserEntity, error) {
	filter := bson.M{"referred_by": referredBy}
	cursor, err := repo.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*odm_entities.UserEntity
	for cursor.Next(ctx) {
		var user odm_entities.UserEntity
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

func (ur *UserRepository) GetReferralCodeByWallet(ctx context.Context, wallet string) (string, error) {
	// Проверяем, что кошелек не пуст
	if wallet == "" {
		return "", errors.New("wallet cannot be empty")
	}

	// Ищем пользователя по кошельку
	var user odm_entities.UserEntity
	err := ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", errors.New("user not found")
		}
		return "", err
	}

	// Возвращаем реферальный код
	return user.ReferralCode, nil
}

func (ur *UserRepository) GetNameByWallet(ctx context.Context, wallet string) (string, error) {
	if wallet == "" {
		return "", errors.New("wallet cannot be empty")
	}

	// Определяем проекцию, чтобы выбрать только поле "name"
	projection := bson.M{"name": 1}

	// Создаем опции с проекцией
	opts := options.FindOne().SetProjection(projection)

	// Ищем пользователя по кошельку с заданной проекцией
	var result struct {
		Name string `bson:"name"`
	}
	err := ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[GetNameByWallet] User not found for wallet: %s", wallet)
			return "", errors.New("user not found")
		}
		log.Printf("[GetNameByWallet] Error fetching user name: %v", err)
		return "", err
	}

	// Логируем найденное имя
	log.Printf("[GetNameByWallet] Retrieved name for wallet %s: %s", wallet, result.Name)

	return result.Name, nil
}

func (ur *UserRepository) GetWalletByReferralCode(ctx context.Context, referralCode string) (string, error) {
	if referralCode == "" {
		return "", errors.New("referral code cannot be empty")
	}

	// Определяем проекцию, чтобы выбрать только поле "wallet"
	projection := bson.M{"wallet": 1}

	// Создаем опции с проекцией
	opts := options.FindOne().SetProjection(projection)

	// Ищем пользователя по реферальному коду с заданной проекцией
	var result struct {
		Wallet string `bson:"wallet"`
	}
	err := ur.Collection.FindOne(ctx, bson.M{"referral_code": referralCode}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[GetWalletByReferralCode] User not found for referral code: %s", referralCode)
			return "", errors.New("user not found")
		}
		log.Printf("[GetWalletByReferralCode] Error fetching wallet: %v", err)
		return "", err
	}

	// Логируем найденный кошелек
	log.Printf("[GetWalletByReferralCode] Retrieved wallet for referral code %s: %s", referralCode, result.Wallet)

	return result.Wallet, nil
}

func (ur *UserRepository) GetFirstNameByWallet(ctx context.Context, wallet string) (string, error) {
	// Проверка на пустой кошелек
	if wallet == "" {
		return "", errors.New("wallet cannot be empty")
	}

	// Определяем проекцию, чтобы выбрать только поле "first_name"
	projection := bson.M{"first_name": 1}

	// Создаем опции с проекцией
	opts := options.FindOne().SetProjection(projection)

	// Ищем пользователя по кошельку с заданной проекцией
	var result struct {
		FirstName string `bson:"first_name"`
	}
	err := ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[GetFirstNameByWallet] User not found for wallet: %s", wallet)
			return "", errors.New("user not found")
		}
		log.Printf("[GetFirstNameByWallet] Error fetching user first name: %v", err)
		return "", err
	}

	// Логируем найденное имя
	log.Printf("[GetFirstNameByWallet] Retrieved first name for wallet %s: %s", wallet, result.FirstName)

	return result.FirstName, nil
}

func (ur *UserRepository) UpdateBalances(ctx context.Context, winnerWallet, loserWallet, tokenType string, winAmount, loseAmount float64) error {
	// Проверяем валидность типа токена
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		return errors.New("invalid token type")
	}

	// Логируем входные данные
	log.Printf("[UpdateBalances] Winner: %s, Loser: %s, Token: %s, WinAmount: %.2f, LoseAmount: %.2f",
		winnerWallet, loserWallet, tokenType, winAmount, loseAmount)

	// Проверяем, что проигравший имеет достаточно средств для списания
	loserBalance, err := ur.GetTokenBalance(ctx, loserWallet, tokenType)
	if err != nil {
		log.Printf("[UpdateBalances] Ошибка получения баланса проигравшего: %v", err)
		return fmt.Errorf("failed to get loser's balance: %w", err)
	}
	if loserBalance < loseAmount {
		log.Printf("[UpdateBalances] Недостаточно средств у проигравшего (Wallet: %s, Balance: %.2f, Required: %.2f)",
			loserWallet, loserBalance, loseAmount)
		return errors.New("insufficient balance for the loser")
	}

	// Выполняем транзакцию обновления балансов
	session, err := ur.Collection.Database().Client().StartSession()
	if err != nil {
		log.Printf("[UpdateBalances] Ошибка создания сессии: %v", err)
		return err
	}
	defer session.EndSession(ctx)

	// Обновление выполняется в транзакции
	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// Списание у проигравшего
		loserUpdate := bson.M{
			"$inc": bson.M{
				tokenType: -loseAmount, // Списание ставки
			},
			"$set": bson.M{
				"updated_at": time.Now(),
			},
		}
		loserFilter := bson.M{"wallet": loserWallet, tokenType: bson.M{"$gte": loseAmount}}
		if _, err := ur.Collection.UpdateOne(sc, loserFilter, loserUpdate); err != nil {
			log.Printf("[UpdateBalances] Ошибка списания средств у проигравшего: %v", err)
			return fmt.Errorf("failed to update loser's balance: %w", err)
		}

		// Начисление победителю
		winnerUpdate := bson.M{
			"$inc": bson.M{
				tokenType: winAmount, // Начисление выигрыша
			},
			"$set": bson.M{
				"updated_at": time.Now(),
			},
		}
		winnerFilter := bson.M{"wallet": winnerWallet}
		if _, err := ur.Collection.UpdateOne(sc, winnerFilter, winnerUpdate); err != nil {
			log.Printf("[UpdateBalances] Ошибка начисления средств победителю: %v", err)
			return fmt.Errorf("failed to update winner's balance: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("[UpdateBalances] Ошибка транзакции: %v", err)
		return fmt.Errorf("transaction failed: %w", err)
	}

	log.Printf("[UpdateBalances] Балансы успешно обновлены: Winner=%s, Loser=%s", winnerWallet, loserWallet)
	return nil
}

func (ur *UserRepository) AddReferralEarnings(ctx context.Context, wallet string, earnings map[string]float64) error {
	log.Printf("[AddReferralEarnings] Updating referral earnings for wallet: %s, earnings: %+v", wallet, earnings)

	updateFields := bson.M{}
	for tokenType, amount := range earnings {
		// Используем $inc для корректного добавления числовых значений
		updateFields["referral_earnings."+tokenType] = amount
	}

	// Выполняем обновление
	result, err := ur.Collection.UpdateOne(
		ctx,
		bson.M{"wallet": wallet},
		bson.M{"$inc": updateFields},
	)

	if err != nil {
		log.Printf("[AddReferralEarnings] MongoDB error while updating: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		log.Printf("[AddReferralEarnings] Wallet %s not found", wallet)
		return errors.New("user not found")
	}

	log.Printf("[AddReferralEarnings] Successfully updated referral earnings for wallet: %s", wallet)
	return nil
}

func (repo *UserRepository) GetReferralEarnings(ctx context.Context, wallet string, tokenType string) (float64, error) {
	log.Printf("[GetReferralEarnings] Fetching referral earnings for wallet %s and token type %s", wallet, tokenType)

	// Проверка на допустимые типы токенов
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}
	if !validTokens[tokenType] {
		log.Printf("[GetReferralEarnings] Invalid token type: %s", tokenType)
		return 0, errors.New("invalid token type")
	}

	// Конструируем pipeline для агрегации
	pipeline := []bson.M{
		{"$match": bson.M{"wallet": wallet}}, // Фильтруем пользователя по wallet
		{"$project": bson.M{
			"total_earnings": bson.M{
				"$ifNull": bson.A{
					"$referral_earnings." + tokenType, 0, // Если earnings отсутствуют, возвращаем 0
				},
			},
		}},
	}

	var result struct {
		TotalEarnings float64 `bson:"total_earnings"`
	}

	// Выполняем агрегацию
	cursor, err := repo.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("[GetReferralEarnings] Error during aggregation: %v", err)
		return 0, err
	}
	defer cursor.Close(ctx)

	// Получаем результат из курсора
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			log.Printf("[GetReferralEarnings] Error decoding result: %v", err)
			return 0, err
		}
		log.Printf("[GetReferralEarnings] Total earnings for %s: %.2f", wallet, result.TotalEarnings)
		return result.TotalEarnings, nil
	}

	log.Printf("[GetReferralEarnings] No earnings found for wallet: %s", wallet)
	return 0, nil // Если данных нет, возвращаем 0
}

func (ur *UserRepository) GetAllReferralEarnings(ctx context.Context, wallet string) (map[string]float64, error) {
	log.Printf("[GetAllReferralEarnings] Fetching all referral earnings for wallet: %s", wallet)

	// Определяем проекцию, чтобы выбрать только поле referral_earnings
	projection := bson.M{"referral_earnings": 1}

	// Опции запроса с проекцией
	opts := options.FindOne().SetProjection(projection)

	// Структура для результата
	var result struct {
		ReferralEarnings map[string]float64 `bson:"referral_earnings"`
	}

	// Выполняем запрос
	err := ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[GetAllReferralEarnings] User not found for wallet: %s", wallet)
			return nil, errors.New("user not found")
		}
		log.Printf("[GetAllReferralEarnings] Error fetching referral earnings: %v", err)
		return nil, err
	}

	// Если поле `referral_earnings` пустое, возвращаем карту с нулями
	if result.ReferralEarnings == nil {
		log.Printf("[GetAllReferralEarnings] No referral earnings found for wallet: %s", wallet)
		return map[string]float64{
			"ton_balance": 0,
			"m5_balance":  0,
			"dfc_balance": 0,
		}, nil
	}

	log.Printf("[GetAllReferralEarnings] Retrieved referral earnings for wallet %s: %+v", wallet, result.ReferralEarnings)
	return result.ReferralEarnings, nil
}

func (ur *UserRepository) GetByName(ctx context.Context, name string) (*odm_entities.UserEntity, error) {
	var user odm_entities.UserEntity

	// Поиск пользователя по имени
	err := ur.Collection.FindOne(ctx, bson.M{"name": name}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) AddPointsForBet(ctx context.Context, wallet string, tokenType string, betAmount float64, isWin bool, gameType string) error {
	log.Printf("[AddPointsForBet] Calculating points for wallet: %s, tokenType: %s, betAmount: %.2f, isWin: %t, gameType: %s", wallet, tokenType, betAmount, isWin, gameType)

	// Проверка валидности токена
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		log.Printf("[AddPointsForBet] Invalid token type: %s", tokenType)
		return errors.New("invalid token type")
	}

	// Вычисление очков за ставку
	points := 0.0
	switch tokenType {
	case "ton_balance":
		switch {
		case betAmount >= 1 && betAmount < 3:
			points += betAmount * 0.4
		case betAmount >= 3 && betAmount < 5:
			points += betAmount * 0.6
		case betAmount >= 5 && betAmount <= 8:
			points += betAmount * 0.8
		case betAmount > 8:
			points += betAmount * 1.0
		}
	case "m5_balance":
		switch {
		case betAmount >= 3 && betAmount < 5:
			points += betAmount * 0.15
		case betAmount >= 5 && betAmount <= 10:
			points += betAmount * 0.225
		case betAmount > 10 && betAmount <= 20:
			points += betAmount * 0.27
		case betAmount > 20:
			points += betAmount * 0.34
		}
	case "dfc_balance":
		switch {
		case betAmount >= 6 && betAmount < 12:
			points += betAmount * 0.06
		case betAmount >= 12 && betAmount <= 24:
			points += betAmount * 0.09
		case betAmount > 24 && betAmount <= 48:
			points += betAmount * 0.11
		case betAmount > 48:
			points += betAmount * 0.13
		}
	}

	// Добавление очков за победу или поражение в зависимости от типа игры
	if gameType == "bot" {
		if isWin {
			points += 0.25 // Победа против бота
		} else {
			points += 0.125 // Поражение против бота
		}
	} else if gameType == "pvp" {
		if isWin {
			points += 0.5 // Победа в PvP
		} else {
			points += 0.25 // Поражение в PvP
		}
	}

	if points == 0 {
		log.Printf("[AddPointsForBet] Bet amount does not qualify for points. Wallet: %s, TokenType: %s, BetAmount: %.2f", wallet, tokenType, betAmount)
		return nil // Нет начислений за ставку
	}

	log.Printf("[AddPointsForBet] Calculated points: %.2f for wallet: %s, tokenType: %s, betAmount: %.2f, isWin: %t, gameType: %s", points, wallet, tokenType, betAmount, isWin, gameType)

	// Обновление очков пользователя
	filter := bson.M{"wallet": wallet}
	update := bson.M{
		"$inc": bson.M{"points": points},
		"$set": bson.M{"updated_at": time.Now()},
	}

	result, err := ur.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("[AddPointsForBet] MongoDB error while updating points: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		log.Printf("[AddPointsForBet] Wallet %s not found or points not updated", wallet)
		return errors.New("user not found")
	}

	log.Printf("[AddPointsForBet] Successfully updated points for wallet: %s. Points added: %.2f", wallet, points)
	return nil
}

func (ur *UserRepository) GetUsersByPointsDescending(ctx context.Context, limit int64, offset int64) ([]odm_entities.UserEntity, error) {
	log.Printf("[GetUsersByPointsDescending] Fetching users sorted by points descending with limit: %d, offset: %d", limit, offset)

	// Настройка параметров для сортировки и пагинации
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "points", Value: -1}}) // Сортировка по points от большего к меньшему
	opts.SetLimit(limit)
	opts.SetSkip(offset)

	// Выполняем поиск пользователей
	cursor, err := ur.Collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		log.Printf("[GetUsersByPointsDescending] Error fetching users: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []odm_entities.UserEntity
	for cursor.Next(ctx) {
		var user odm_entities.UserEntity
		if err := cursor.Decode(&user); err != nil {
			log.Printf("[GetUsersByPointsDescending] Error decoding user: %v", err)
			return nil, err
		}
		users = append(users, user)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("[GetUsersByPointsDescending] Cursor error: %v", err)
		return nil, err
	}

	log.Printf("[GetUsersByPointsDescending] Successfully fetched users sorted by points")
	return users, nil
}

func (ur *UserRepository) GetByTgID(ctx context.Context, tgID string) (*odm_entities.UserEntity, error) {
	log.Printf("[GetByTgID] Поиск пользователя с TgID: %s", tgID)

	var user odm_entities.UserEntity
	filter := bson.M{"tgid": tgID}

	err := ur.Collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[GetByTgID] Пользователь с TgID %s не найден", tgID)
			return nil, errors.New("user not found")
		}
		log.Printf("[GetByTgID] Ошибка при поиске пользователя: %v", err)
		return nil, fmt.Errorf("error fetching user by TgID: %v", err)
	}

	log.Printf("[GetByTgID] Пользователь найден: %+v", user)
	return &user, nil
}

func (ur *UserRepository) GetPointsByWallet(ctx context.Context, wallet string) (float64, error) {
	log.Printf("[GetPointsByWallet] Fetching points for wallet: %s", wallet)

	// Определяем проекцию, чтобы выбрать только поле points
	projection := bson.M{"points": 1}

	// Создаем опции с проекцией
	opts := options.FindOne().SetProjection(projection)

	var result struct {
		Points float64 `bson:"points"`
	}

	// Ищем пользователя по кошельку с заданной проекцией
	err := ur.Collection.FindOne(ctx, bson.M{"wallet": wallet}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[GetPointsByWallet] User not found for wallet: %s", wallet)
			return 0, errors.New("user not found")
		}
		log.Printf("[GetPointsByWallet] Error fetching points: %v", err)
		return 0, err
	}

	log.Printf("[GetPointsByWallet] Successfully fetched points for wallet: %s. Points: %.2f", wallet, result.Points)
	return result.Points, nil
}

func (ur *UserRepository) ApplyPromoCodeRewards(ctx context.Context, wallet string, tokenType string, amount float64) error {
	log.Printf("[ApplyPromoCodeRewards] Applying promocode rewards to wallet: %s, type: %s, amount: %.2f", wallet, tokenType, amount)

	switch tokenType {
	case "ton_balance", "m5_balance", "dfc_balance":
		// Handle token rewards
		err := ur.AddTokens(ctx, wallet, map[string]float64{tokenType: amount})
		if err != nil {
			log.Printf("[ApplyPromoCodeRewards] Error applying token reward: %v", err)
			return err
		}

	case "cube":
		// Handle cube rewards
		err := ur.AddCubes(ctx, wallet, int(amount))
		if err != nil {
			log.Printf("[ApplyPromoCodeRewards] Error applying cube reward: %v", err)
			return err
		}

	default:
		log.Printf("[ApplyPromoCodeRewards] Invalid reward type: %s", tokenType)
		return errors.New("invalid reward type")
	}

	log.Printf("[ApplyPromoCodeRewards] Rewards applied successfully to wallet: %s", wallet)
	return nil
}
