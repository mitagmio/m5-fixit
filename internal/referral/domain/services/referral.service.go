package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"

	"github.com/Peranum/tg-dice/internal/user/infrastructure/odm-entities"
	"github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
)

type ReferralService struct {
	UserRepo *repositories.UserRepository
}

// NewReferralService создает новый ReferralService
func NewReferralService(userRepo *repositories.UserRepository) *ReferralService {
	return &ReferralService{
		UserRepo: userRepo,
	}
}

// Генерация реферального кода
func (rs *ReferralService) GenerateReferralCode(TgID string) (string, error) {
	const length = 6

	if TgID == "" {
		return "", errors.New("wallet cannot be empty")
	}

	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", errors.New("failed to generate random bytes")
	}

	referralCode := hex.EncodeToString(bytes)
	if len(referralCode) > length {
		referralCode = referralCode[:length] // Truncate to ensure exact length
	}

	// Добавляем последние 2 символа кошелька
	if len(TgID) > 2 {
		referralCode += TgID[len(TgID)-2:]
	}

	return referralCode, nil
}

// Получение рефералов по уровню
func (rs *ReferralService) GetReferralsByLevel(ctx context.Context, referralCode string, level int) ([]*odm_entities.UserEntity, error) {
	if level <= 0 {
		return nil, errors.New("level must be greater than zero")
	}

	currentReferrals := []*odm_entities.UserEntity{}
	nextReferrals := []*odm_entities.UserEntity{}

	// Начальный уровень
	referrals, err := rs.UserRepo.GetUsersByReferredBy(ctx, referralCode)
	if err != nil {
		return nil, err
	}
	currentReferrals = referrals

	if level == 1 {
		return currentReferrals, nil
	}

	// Уровни от 2 до N
	for currentLevel := 2; currentLevel <= level; currentLevel++ {
		nextReferrals = []*odm_entities.UserEntity{}
		for _, referral := range currentReferrals {
			subReferrals, err := rs.UserRepo.GetUsersByReferredBy(ctx, referral.ReferralCode)
			if err != nil {
				return nil, err
			}
			nextReferrals = append(nextReferrals, subReferrals...)
		}
		currentReferrals = nextReferrals
		if currentLevel == level {
			return currentReferrals, nil
		}
	}

	return currentReferrals, nil
}

func (rs *ReferralService) GetTotalReferrals(ctx context.Context, referralCode string) (int, error) {
	var totalReferrals int

	currentReferrals, err := rs.UserRepo.GetUsersByReferredBy(ctx, referralCode)
	if err != nil {
		return 0, err
	}

	totalReferrals += len(currentReferrals)

	for _, referral := range currentReferrals {
		subTotal, err := rs.GetTotalReferrals(ctx, referral.ReferralCode)
		if err != nil {
			return 0, err
		}
		totalReferrals += subTotal
	}

	return totalReferrals, nil
}

// Получение рефералов по уровням с их именами
func (rs *ReferralService) GetReferralsByLevels(ctx context.Context, referralCode string) (map[string][]string, error) {
	levels := map[string][]string{
		"level1": {},
		"level2": {},
		"level3": {},
	}

	// Первый уровень
	level1Referrals, err := rs.GetReferralsByLevel(ctx, referralCode, 1)
	if err != nil {
		return nil, err
	}
	for _, referral := range level1Referrals {
		levels["level1"] = append(levels["level1"], referral.Name)
	}

	// Второй уровень
	for _, referral := range level1Referrals {
		level2Referrals, err := rs.GetReferralsByLevel(ctx, referral.ReferralCode, 1)
		if err != nil {
			return nil, err
		}
		for _, subReferral := range level2Referrals {
			levels["level2"] = append(levels["level2"], subReferral.Name)
		}
	}

	// Третий уровень
	for _, referral := range level1Referrals {
		level2Referrals, err := rs.GetReferralsByLevel(ctx, referral.ReferralCode, 1)
		if err != nil {
			return nil, err
		}
		for _, subReferral := range level2Referrals {
			level3Referrals, err := rs.GetReferralsByLevel(ctx, subReferral.ReferralCode, 1)
			if err != nil {
				return nil, err
			}
			for _, thirdLevelReferral := range level3Referrals {
				levels["level3"] = append(levels["level3"], thirdLevelReferral.Name)
			}
		}
	}

	return levels, nil
}

func (rs *ReferralService) GetReferralsByWallet(ctx context.Context, wallet string) (map[string][]string, error) {
	user, err := rs.UserRepo.GetByWallet(ctx, wallet)
	if err != nil {
		return nil, errors.New("failed to fetch user by wallet")
	}

	if user.ReferralCode == "" {
		return nil, errors.New("user does not have a referral code")
	}

	return rs.GetReferralsByLevels(ctx, user.ReferralCode)
}

func (rs *ReferralService) DistributeReferralReward(ctx context.Context, wallet string, rewardAmount float64, tokenType string) error {
	log.Printf("[DistributeReferralReward] Starting reward distribution. Wallet=%s, RewardAmount=%.2f, TokenType=%s", wallet, rewardAmount, tokenType)

	if rewardAmount <= 0 {
		log.Printf("[DistributeReferralReward] Invalid reward amount: %.2f", rewardAmount)
		return errors.New("reward amount must be greater than zero")
	}

	// Проверяем, что переданный tokenType валиден
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}
	if !validTokens[tokenType] {
		log.Printf("[DistributeReferralReward] Invalid token type: %s", tokenType)
		return errors.New("invalid token type")
	}

	// Текущий пользователь, который получил награду
	currentWallet := wallet

	// Коэффициенты распределения награды
	rewardDistribution := []float64{0.05, 0.02, 0.01} // 5%, 2%, 1% для 1-го, 2-го и 3-го уровней

	for level, percentage := range rewardDistribution {
		log.Printf("[DistributeReferralReward] Processing level %d for wallet %s", level+1, currentWallet)

		// Получаем пользователя по текущему кошельку
		user, err := rs.UserRepo.GetByWallet(ctx, currentWallet)
		if err != nil {
			log.Printf("[DistributeReferralReward] Failed to fetch user for wallet %s at level %d: %v", currentWallet, level+1, err)
			return nil // Прерываем распределение на этом уровне
		}

		// Логируем данные найденного пользователя
		log.Printf("[DistributeReferralReward] Found user at level %d: %+v", level+1, user)

		// Проверяем наличие реферального кода
		if user.ReferredBy == "" {
			log.Printf("[DistributeReferralReward] User %s does not have a referrer. Stopping reward distribution at level %d", currentWallet, level+1)
			break
		}

		// Получаем кошелек реферера на основе реферального кода
		referrerWallet, err := rs.UserRepo.GetWalletByReferralCode(ctx, user.ReferredBy)
		if err != nil {
			log.Printf("[DistributeReferralReward] Failed to get wallet for referral code %s at level %d: %v", user.ReferredBy, level+1, err)
			return err
		}

		// Вычисляем сумму награды для этого уровня
		rewardForLevel := rewardAmount * percentage
		log.Printf("[DistributeReferralReward] Calculated reward for level %d: %.2f %s", level+1, rewardForLevel, tokenType)

		// Обновляем баланс реферера
		err = rs.UserRepo.AddTokens(ctx, referrerWallet, map[string]float64{tokenType: rewardForLevel})
		if err != nil {
			log.Printf("[DistributeReferralReward] Failed to distribute reward to wallet %s at level %d: %v", referrerWallet, level+1, err)
			return err
		}

		// Обновляем реферальные начисления реферера
		// Обновляем реферальные начисления реферера
		err = rs.UserRepo.AddReferralEarnings(ctx, referrerWallet, map[string]float64{
			tokenType: rewardForLevel,
		})
		if err != nil {
			log.Printf("[DistributeReferralReward] Failed to update referral earnings for wallet %s at level %d: %v", referrerWallet, level+1, err)
			return err
		}

		log.Printf("[DistributeReferralReward] Successfully distributed %.2f %s to wallet %s at level %d and updated referral earnings", rewardForLevel, tokenType, referrerWallet, level+1)

		// Переходим к следующему уровню
		currentWallet = referrerWallet
	}

	log.Printf("[DistributeReferralReward] Reward distribution completed for wallet %s", wallet)
	return nil
}
