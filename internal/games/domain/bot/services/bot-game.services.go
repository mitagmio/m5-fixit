package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/Peranum/tg-dice/internal/games/domain/history/services" // Сервис для сохранения игры
	"github.com/Peranum/tg-dice/internal/games/infrastructure/bot/entity"
	botRepos "github.com/Peranum/tg-dice/internal/games/infrastructure/bot/repositories"
	refService "github.com/Peranum/tg-dice/internal/referral/domain/services"
	userRepos "github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
	"log"
	"math/rand"
	"time"
)

type DiceGameResult struct {
	UserScore   int    `json:"user_score"`
	BotScore    int    `json:"bot_score"`
	Winner      string `json:"winner"`
	Rounds      int    `json:"rounds"`
	TargetScore int    `json:"target_score"`
	UserRolls   []int  `json:"user_rolls"`
	BotRolls    []int  `json:"bot_rolls"`
}

type BotGameService struct {
	BotRepo     *botRepos.BotRepository
	UserRepo    *userRepos.UserRepository
	GameService *services.GameService
	RefService  *refService.ReferralService // Убедитесь, что поле объявлено
}

func NewBotGameService(
	botRepo *botRepos.BotRepository,
	userRepo *userRepos.UserRepository,
	gameService *services.GameService,
	refService *refService.ReferralService, // Передаем refService как аргумент
) *BotGameService {
	return &BotGameService{
		BotRepo:     botRepo,
		UserRepo:    userRepo,
		GameService: gameService,
		RefService:  refService, // Инициализируем поле RefService
	}
}

func (gs *BotGameService) PlayDiceGame(ctx context.Context, wallet string, tokenType string, betAmount float64, targetScore int) (map[string]interface{}, error) {
	if targetScore < 15 || targetScore > 45 {
		log.Printf("[PlayDiceGame] Invalid targetScore=%d", targetScore)
		return nil, errors.New("target score must be between 15 and 45")
	}

	botBalance, err := gs.BotRepo.GetTokenBalance(ctx, tokenType)
	if err != nil {
		log.Printf("[PlayDiceGame] Failed to retrieve bot balance: %v", err)
		return nil, errors.New("failed to retrieve bot balance")
	}

	hasBalance, err := gs.UserRepo.HasSufficientBalance(ctx, wallet, tokenType, betAmount)
	if err != nil {
		log.Printf("[PlayDiceGame] Error checking user balance: %v", err)
		return nil, errors.New("failed to check user balance")
	}
	if !hasBalance {
		log.Printf("[PlayDiceGame] User does not have sufficient balance. Wallet=%s, TokenType=%s, Bet=%.2f, BotBalance=%.2f",
			wallet, tokenType, betAmount, botBalance)
		return nil, errors.New("user does not have sufficient balance")
	}

	// Получаем имя пользователя по кошельку (предположим, что функция возвращает first_name)
	player1FirstName, err := gs.UserRepo.GetFirstNameByWallet(ctx, wallet)
	if err != nil {
		log.Printf("[PlayDiceGame] Failed to retrieve user first name for wallet=%s: %v", wallet, err)
		return nil, errors.New("failed to retrieve user first name")
	}
	player1Name := player1FirstName // Присваиваем имя игрока

	player2Name := "Bob"

	// Определяем, близок ли баланс бота к нулю
	botLowBalance := botBalance <= betAmount*3 || botBalance < 10
	log.Printf("[PlayDiceGame] botLowBalance=%t (botBalance=%.2f, bet=%.2f)", botLowBalance, botBalance, betAmount)

	userScore, botScore := 0, 0
	rounds := 0
	var roundsDetails []map[string]interface{}

	rand.Seed(time.Now().UnixNano())

	for userScore < targetScore && botScore < targetScore {
		rounds++

		var userRoll1, userRoll2 int
		if botLowBalance {
			userRoll1 = rand.Intn(3) + 1
			userRoll2 = rand.Intn(3) + 1
		} else {
			userRoll1 = rand.Intn(6) + 1
			userRoll2 = rand.Intn(6) + 1
		}
		userRoundScore := userRoll1 + userRoll2
		if userRoll1 == userRoll2 {
			userRoundScore++
		}

		var botRoll1, botRoll2 int
		botRoll1 = rand.Intn(6) + 1 // Обычные броски в диапазоне [1, 6]
		botRoll2 = rand.Intn(6) + 1

		botRoundScore := botRoll1 + botRoll2
		if botRoll1 == botRoll2 {
			botRoundScore++
		}

		userScore += userRoundScore
		botScore += botRoundScore

		// Присваиваем результат работы append обратно в переменную
		roundsDetails = append(roundsDetails, map[string]interface{}{
			"round":            rounds,
			"bot_rolls":        []int{botRoll1, botRoll2},
			"bot_round_score":  botRoundScore,
			"user_rolls":       []int{userRoll1, userRoll2},
			"user_round_score": userRoundScore,
		})
	}

	if userScore == botScore {
		for userScore == botScore {
			rounds++
			log.Printf("[PlayDiceGame] User and bot scores are equal. Adding a tie-breaking round.")

			// Повторяем раунд для определения победителя
			userRoll1, userRoll2 := rand.Intn(6)+1, rand.Intn(6)+1
			botRoll1, botRoll2 := rand.Intn(6)+1, rand.Intn(6)+1

			userRoundScore := userRoll1 + userRoll2
			botRoundScore := botRoll1 + botRoll2

			userScore += userRoundScore
			botScore += botRoundScore

			roundsDetails = append(roundsDetails, map[string]interface{}{
				"round":            rounds,
				"bot_rolls":        []int{botRoll1, botRoll2},
				"bot_round_score":  botRoundScore,
				"user_rolls":       []int{userRoll1, userRoll2},
				"user_round_score": userRoundScore,
			})
		}
	}

	// Определение победителя
	var winner string
	if userScore >= targetScore && userScore > botScore {
		winner = "user"
	} else {
		winner = "bot"
	}

	log.Printf("[PlayDiceGame] Game ended: winner=%s, userScore=%d, botScore=%d, botLowBalance=%t", winner, userScore, botScore, botLowBalance)

	// Обновление балансов и подсчёт заработка для сохранения
	var player1Earnings, player2Earnings float64

	if winner == "user" {
		isWin := true
		if err := gs.BotRepo.AddTokenBalance(ctx, tokenType, -betAmount); err != nil {
			log.Printf("[PlayDiceGame] Failed to update bot balance after user win: %v", err)
			return nil, errors.New("failed to update bot balance")
		}
		player1Earnings = betAmount
		player2Earnings = -betAmount
		log.Printf("[PlayDiceGame] User won. Bot balance decreased by %.2f", betAmount)
		err := gs.UserRepo.AddPointsForBet(ctx, wallet, tokenType, betAmount, isWin, "bot")
		if err != nil {
			log.Printf("Failed to add points: %v", err)
		}
		if err := gs.UserRepo.AddTokens(ctx, wallet, map[string]float64{tokenType: betAmount}); err != nil {
			log.Printf("[PlayDiceGame] Failed to update user balance after user lose: %v", err)
			return nil, errors.New("failed to update user balance")
		}
	} else {
		isWin := false
		if err := gs.UserRepo.AddTokens(ctx, wallet, map[string]float64{tokenType: -betAmount}); err != nil {
			log.Printf("[PlayDiceGame] Failed to update user balance after user lose: %v", err)
			return nil, errors.New("failed to update user balance")
		}
		if err := gs.BotRepo.AddTokenBalance(ctx, tokenType, betAmount); err != nil {
			log.Printf("[PlayDiceGame] Failed to update bot balance after user lose: %v", err)
			return nil, errors.New("failed to update bot balance")
		}
		player1Earnings = -betAmount
		player2Earnings = betAmount * 2
		log.Printf("[PlayDiceGame] User lost. User balance decreased by %.2f, bot balance increased by %.2f", betAmount, betAmount)

		// Распределение награды рефералам
		if err := gs.RefService.DistributeReferralReward(ctx, wallet, betAmount*2, tokenType); err != nil {
			log.Printf("[PlayDiceGame] Failed to distribute referral reward: %v", err)
			return nil, errors.New("failed to distribute referral reward")
		}
		log.Printf("[PlayDiceGame] Referral reward distributed for wallet=%s, rewardAmount=%.2f, tokenType=%s", wallet, betAmount, tokenType)
		err := gs.UserRepo.AddPointsForBet(ctx, wallet, tokenType, betAmount, isWin, "bot")
		if err != nil {
			log.Printf("Failed to add points for loss: %v", err)
		}
	}

	// Сохранение результатов игры
	err = gs.GameService.SaveGame(
		ctx,
		player1Name,     // Имя пользователя
		player2Name,     // Имя бота
		userScore,       // Очки игрока
		botScore,        // Очки бота
		winner,          // Победитель
		player1Earnings, // Заработок игрока
		player2Earnings, // Заработок бота
		tokenType,       // Тип токена
		betAmount,       // Сумма ставки
		wallet,          // Кошелёк игрока
		"Bob",           // Имя бота
	)
	if err != nil {
		log.Printf("Error saving game results: %v", err)
	}

	// Формирование результата игры
	result := map[string]interface{}{
		"winner":          winner,
		"user_score":      userScore,
		"bot_score":       botScore,
		"rounds_played":   rounds,
		"rounds_details":  roundsDetails,
		"bot_low_balance": botLowBalance,
		"token_type":      tokenType,
		"bet_amount":      betAmount,
		"player_name":     player1Name, // Добавляем имя игрока
	}

	return result, nil
}

func (gs *BotGameService) SimulateDiceGameForUserWin(ctx context.Context, wallet string) (string, error) {
	// Фиксируем targetScore как 25
	targetScore := 25

	// Инициализация переменных для игры
	userScore, botScore := 0, 0
	var userRolls, botRolls []int
	rounds := 0

	// Строка для накопления результатов каждого раунда
	var roundsResult string

	rand.Seed(time.Now().UnixNano())

	// Игровой цикл: продолжаем играть, пока кто-то не достигнет targetScore
	for userScore < targetScore && botScore < targetScore {
		rounds++

		// Роллы пользователя с бонусом +1 на каждую кость
		userRoll1 := rand.Intn(6) + 1
		userRoll2 := rand.Intn(6) + 1
		userRoundScore := userRoll1 + userRoll2 + 1

		// Если выпал дубль, добавляем еще одно очко
		if userRoll1 == userRoll2 {
			userRoundScore++
		}

		// Роллы бота без бонуса (имитация слабой игры бота)
		// Результат бота будет всегда хуже результата пользователя
		botRoll1 := rand.Intn(5) + 1
		botRoll2 := rand.Intn(3) + 1
		botRoundScore := botRoll1 + botRoll2 // Бот всегда будет иметь минимальный результат

		// Если выпал дубль у бота, добавляем еще одно очко
		if botRoll1 == botRoll2 {
			botRoundScore++
		}

		// Накопление очков
		userScore += userRoundScore
		botScore += botRoundScore

		userRolls = append(userRolls, userRoundScore)
		botRolls = append(botRolls, botRoundScore)

		log.Printf("[SimulateDiceGameForUserWin] Round %d: userScore=%d(+%d), botScore=%d(+%d)", rounds, userScore, userRoundScore, botScore, botRoundScore)

		// Добавление описания раунда на английском
		roundsResult += fmt.Sprintf("Round %d Bot %d %d User %d %d ", rounds, botRoll1, botRoll2, userRoll1, userRoll2)
	}

	// Обеспечиваем, что пользователь точно выиграет
	userScore = targetScore
	botScore = targetScore - 1 // Устанавливаем счет бота ниже, чтобы пользователь выиграл

	// Итоговый результат
	winner := "user"
	log.Printf("[SimulateDiceGameForUserWin] Game ended: winner=%s, userScore=%d, botScore=%d", winner, userScore, botScore)

	// Дописываем результат игры
	roundsResult += fmt.Sprintf("Winner: %s", winner)

	// Если пользователь выиграл, начисляем 3 куба
	if winner == "user" {
		// Добавление кубов пользователю через метод AddCubes
		err := gs.UserRepo.AddCubes(ctx, wallet, 3)
		if err != nil {
			log.Printf("[SimulateDiceGameForUserWin] Failed to add cubes to user: %v", err)
			return "", errors.New("failed to add cubes to user")
		}
		log.Printf("[SimulateDiceGameForUserWin] User won. 3 cubes added to user wallet %s", wallet)
	}

	return roundsResult, nil
}

func (bgs *BotGameService) SubtractTokensFromBotBalance(ctx context.Context, tokenType string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	// Вызов метода репозитория для уменьшения баланса
	err := bgs.BotRepo.SubtractTokenBalance(ctx, tokenType, amount)
	if err != nil {
		log.Printf("[SubtractTokensFromBotBalance] Error subtracting tokens: %v", err)
		return err
	}

	log.Printf("[SubtractTokensFromBotBalance] Successfully subtracted %.2f from %s balance", amount, tokenType)
	return nil
}

func (bgs *BotGameService) AddTokensToBotBalance(ctx context.Context, tokenType string, amount float64) error {
	// Логирование для отладки
	log.Printf("[AddTokensToBotBalance] Начинается добавление %.2f токенов типа %s к балансу бота", amount, tokenType)

	// Проверяем корректность типа токена
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		log.Printf("[AddTokensToBotBalance] Ошибка: недопустимый тип токена %s", tokenType)
		return errors.New("invalid token type")
	}

	// Вызываем метод репозитория для добавления токенов
	err := bgs.BotRepo.AddTokenBalance(ctx, tokenType, amount)
	if err != nil {
		log.Printf("[AddTokensToBotBalance] Ошибка при добавлении токенов к балансу бота: %v", err)
		return errors.New("failed to add tokens to bot balance")
	}

	log.Printf("[AddTokensToBotBalance] Успешно добавлено %.2f токенов типа %s к балансу бота", amount, tokenType)
	return nil
}

func (bgs *BotGameService) InitializeBotBalance(ctx context.Context, tonBalance, m5Balance, dfcBalance float64) error {
	// Проверка на существующий баланс
	_, err := bgs.BotRepo.GetTokenBalance(ctx, "ton_balance")
	if err == nil {
		return errors.New("bot balance already exists")
	}

	// Создаем баланс
	err = bgs.BotRepo.CreateBotBalance(ctx, tonBalance, m5Balance, dfcBalance)
	if err != nil {
		return errors.New("failed to create bot balance: " + err.Error())
	}

	return nil
}

// GetBotTokenBalance - Получить баланс токенов бота для определённого типа токена.
func (bgs *BotGameService) GetBotBalance(ctx context.Context) (entities.BotBalanceEntity, error) {
	return bgs.BotRepo.GetBotBalance(ctx)
}

func (bgs *BotGameService) GetTokenBalance(ctx context.Context, tokenType string) (float64, error) {
	return bgs.BotRepo.GetTokenBalance(ctx, tokenType)
}
