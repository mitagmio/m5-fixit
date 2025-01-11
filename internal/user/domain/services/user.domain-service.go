package services

import (
	"context"
	"errors"
	"log"
	"time"

	referral "github.com/Peranum/tg-dice/internal/referral/domain/services"
	"github.com/Peranum/tg-dice/internal/user/domain/entities"
	"github.com/Peranum/tg-dice/internal/user/domain/mapper"
	"github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
	"go.mongodb.org/mongo-driver/bson"
)

type UserDomainService struct {
	UserRepo        *repositories.UserRepository
	ReferralService *referral.ReferralService
}

func NewUserDomainService(userRepo *repositories.UserRepository, referralService *referral.ReferralService) *UserDomainService {
	return &UserDomainService{
		UserRepo:        userRepo,
		ReferralService: referralService,
	}
}

func (ds *UserDomainService) CreateUser(ctx context.Context, user *entities.User) (*entities.User, error) {
	log.Printf("[CreateUser] Generating referral code for TgID=%s", user.TgID)
	referralCode, err := ds.ReferralService.GenerateReferralCode(user.TgID)
	if err != nil {
		log.Printf("[CreateUser] Failed to generate referral code: %v", err)
		return nil, errors.New("failed to generate referral code")
	}
	user.ReferralCode = referralCode

	if user.Language == "" {
		user.Language = "RU" // Значение по умолчанию
	}

	log.Printf("[CreateUser] Mapping to ODM structure: %+v", user)
	odmEntity, err := mapper.ToODM(user)
	if err != nil {
		log.Printf("[CreateUser] Mapping error: %v", err)
		return nil, errors.New("invalid user mapping")
	}

	log.Printf("[CreateUser] Creating user in repository")
	createdOdmEntity, err := ds.UserRepo.Create(ctx, odmEntity)
	if err != nil {
		if err.Error() == "user with this TgID already exists" {
			log.Printf("[CreateUser] User already exists, fetching existing user")
			existingUserEntity, err := ds.UserRepo.GetByTgID(ctx, user.TgID)
			if err != nil {
				log.Printf("[CreateUser] Error fetching existing user: %v", err)
				return nil, err
			}
			existingUser := mapper.ToDomain(existingUserEntity)
			log.Printf("[CreateUser] Returning existing user: %+v", existingUser)
			return existingUser, nil
		}
		log.Printf("[CreateUser] Error creating user: %v", err)
		return nil, err
	}

	createdUser := mapper.ToDomain(createdOdmEntity)
	log.Printf("[CreateUser] User created successfully: %+v", createdUser)
	return createdUser, nil
}

func (ds *UserDomainService) GetTokenBalance(ctx context.Context, wallet string, tokenType string) (float64, error) {
	// Проверяем, передан ли валидный кошелек и токен
	if wallet == "" {
		return 0, errors.New("wallet cannot be empty")
	}
	if tokenType == "" {
		return 0, errors.New("token type cannot be empty")
	}

	// Вызов метода репозитория
	return ds.UserRepo.GetTokenBalance(ctx, wallet, tokenType)
}

// GetUserByID возвращает пользователя по ID
func (ds *UserDomainService) GetUserByID(ctx context.Context, id string) (*entities.User, error) {
	odmEntity, err := ds.UserRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if odmEntity == nil {
		return nil, errors.New("user not found")
	}

	return mapper.ToDomain(odmEntity), nil
}

func (ds *UserDomainService) PatchUserByTgID(ctx context.Context, tgid string, updateData map[string]interface{}) error {
	// Проверяем, что map не пустая
	if len(updateData) == 0 {
		return errors.New("no fields provided for update")
	}

	return ds.UserRepo.UpdateByTgID(ctx, tgid, updateData)
}

// DeleteUser удаляет пользователя по ID
func (ds *UserDomainService) DeleteUser(ctx context.Context, id string) error {
	return ds.UserRepo.Delete(ctx, id)
}

// ListUsers возвращает список пользователей с пагинацией
func (ds *UserDomainService) ListUsers(ctx context.Context, limit int64, offset int64) ([]entities.User, error) {
	odmEntities, err := ds.UserRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	domainUsers := make([]entities.User, len(odmEntities))
	for i, odmEntity := range odmEntities {
		domainUsers[i] = *mapper.ToDomain(&odmEntity)
	}

	return domainUsers, nil
}

// GetUserByWallet возвращает пользователя по кошельку
func (ds *UserDomainService) GetUserByWallet(ctx context.Context, wallet string) (*entities.User, error) {
	odmEntity, err := ds.UserRepo.GetByWallet(ctx, wallet)
	if err != nil {
		return nil, err
	}

	if odmEntity == nil {
		return nil, errors.New("user not found")
	}

	return mapper.ToDomain(odmEntity), nil
}

func (ds *UserDomainService) GetUserBalances(ctx context.Context, wallet string) (map[string]interface{}, error) {
	return ds.UserRepo.GetUserBalances(ctx, wallet)
}

func (ds *UserDomainService) GetReferralCodeByWallet(ctx context.Context, wallet string) (string, error) {
	// Проверяем, что кошелек не пуст
	if wallet == "" {
		return "", errors.New("wallet cannot be empty")
	}

	// Вызов репозитория для получения реферального кода
	referralCode, err := ds.UserRepo.GetReferralCodeByWallet(ctx, wallet)
	if err != nil {
		return "", err
	}

	return referralCode, nil
}

// GetUserReferralEarnings возвращает все реферальные earnings пользователя по его wallet
func (ds *UserDomainService) GetUserReferralEarnings(ctx context.Context, wallet string) (map[string]float64, error) {
	// Проверяем, что кошелек не пуст
	if wallet == "" {
		return nil, errors.New("wallet cannot be empty")
	}

	// Вызов метода репозитория
	earnings, err := ds.UserRepo.GetAllReferralEarnings(ctx, wallet)
	if err != nil {
		return nil, err
	}

	return earnings, nil
}
func (ds *UserDomainService) GetUserByName(ctx context.Context, name string) (*entities.User, error) {
	// Вызов репозитория для поиска пользователя
	odmEntity, err := ds.UserRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return mapper.ToDomain(odmEntity), nil
}

// GetUserPointsByWallet возвращает текущие очки пользователя по его кошельку
func (ds *UserDomainService) GetUserPointsByWallet(ctx context.Context, wallet string) (float64, error) {
	log.Printf("[GetUserPointsByWallet] Fetching points for wallet: %s", wallet)

	// Проверяем, что кошелек не пуст
	if wallet == "" {
		return 0, errors.New("wallet cannot be empty")
	}

	// Вызов функции репозитория
	points, err := ds.UserRepo.GetPointsByWallet(ctx, wallet)
	if err != nil {
		log.Printf("[GetUserPointsByWallet] Error fetching points for wallet: %s, error: %v", wallet, err)
		return 0, err
	}

	log.Printf("[GetUserPointsByWallet] Points for wallet: %s = %.2f", wallet, points)
	return points, nil
}

// GetUsersSortedByPoints возвращает список пользователей, отсортированных по очкам, с пагинацией
func (ds *UserDomainService) GetUsersSortedByPoints(ctx context.Context, limit int64, offset int64) ([]entities.User, error) {
	log.Printf("[GetUsersSortedByPoints] Fetching users sorted by points with limit: %d, offset: %d", limit, offset)

	// Вызов функции репозитория
	odmUsers, err := ds.UserRepo.GetUsersByPointsDescending(ctx, limit, offset)
	if err != nil {
		log.Printf("[GetUsersSortedByPoints] Error fetching users: %v", err)
		return nil, err
	}

	// Преобразуем ODM объекты в доменные
	domainUsers := make([]entities.User, len(odmUsers))
	for i, odmUser := range odmUsers {
		domainUsers[i] = *mapper.ToDomain(&odmUser)
	}

	log.Printf("[GetUsersSortedByPoints] Successfully fetched users sorted by points")
	return domainUsers, nil
}

// GetDailyGamesHistory возвращает историю игр за текущие сутки
func (ds *UserDomainService) GetDailyGamesHistory(ctx context.Context, wallet string) ([]entities.GameHistoryItem, []entities.GameHistoryItem, error) {
	startOfDay := time.Now().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	filter := bson.M{
		"time_played": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
		"player1_wallet": wallet,
		"$or": []bson.M{
			{"$and": []bson.M{
				{"bet_amount": bson.M{"$gte": 0.5}},
				{"token_type": "ton_balance"},
			}},
			{"$and": []bson.M{
				{"bet_amount": bson.M{"$gte": 5}},
				{"token_type": "m5_balance"},
			}},
			{"$and": []bson.M{
				{"bet_amount": bson.M{"$gte": 10}},
				{"token_type": "dfc_balance"},
			}},
		},
	}

	return ds.UserRepo.GetDailyGames(ctx, filter)
}

// CheckAndGiveDailyBonus проверяет и начисляет ежедневный бонус
func (ds *UserDomainService) CheckAndGiveDailyBonus(ctx context.Context, wallet string) error {
	pvpGames, botGames, err := ds.GetDailyGamesHistory(ctx, wallet)
	if err != nil {
		return err
	}

	totalGames := len(pvpGames) + len(botGames)
	if totalGames >= 10 {
		// Проверяем, не был ли уже начислен бонус сегодня
		bonusGiven, err := ds.UserRepo.CheckDailyBonusStatus(ctx, wallet)
		if err != nil {
			return err
		}

		if !bonusGiven {
			// Начисляем 2 кубика
			if err := ds.UserRepo.AddCubes(ctx, wallet, 2); err != nil {
				return err
			}
			// Отмечаем, что бонус начислен
			if err := ds.UserRepo.SetDailyBonusGiven(ctx, wallet); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ds *UserDomainService) CheckDailyBonusStatus(ctx context.Context, wallet string) (bool, error) {
	// Получаем историю игр пользователя за сегодня
	dailyGames, _, err := ds.GetDailyGamesHistory(ctx, wallet)
	if err != nil {
		return false, err
	}

	// Если есть игры сегодня, значит бонус уже получен
	return len(dailyGames) > 0, nil
}
