package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	slotEntities "github.com/Peranum/tg-dice/internal/games/infrastructure/slots/entities"
	slotRepositories "github.com/Peranum/tg-dice/internal/games/infrastructure/slots/repositories"
	userRepositories "github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
)

const (
	JackpotCombo   = "777"
	MediumWinCombo = "111, 222, 333, 444, 555, 666" // Пример для MediumWinCombo
	SmallWinCombo  = "717, 727, 737, 747, 757, 767, 776, 771, 772, 773, 774, 775, 776, 177, 277, 377, 477, 577, 677"
	NoWinCombo     = "NoWin"
	MinTonBet      = 0.1
	MaxTonBet      = 10.0
	MinCubeBet     = 1
	MaxCubeBet     = 40
	CubeToTonRate  = 0.25
)

// SlotGameService - Сервис для работы с играми слотов.
type SlotGameService struct {
	SlotRepository     *slotRepositories.SlotGameRepository
	UserRepo           *userRepositories.UserRepository
	CompanyBalanceRepo *slotRepositories.SlotsBalanceRepository // Репозиторий для работы с балансом компании
}

// NewSlotGameService - Конструктор для создания нового SlotGameService.
func NewSlotGameService(
	slotRepo *slotRepositories.SlotGameRepository,
	userRepo *userRepositories.UserRepository,
	companyBalanceRepo *slotRepositories.SlotsBalanceRepository,
) *SlotGameService {
	// Инициализируем генератор случайных чисел один раз
	rand.Seed(time.Now().UnixNano())
	return &SlotGameService{
		SlotRepository:     slotRepo,
		UserRepo:           userRepo,
		CompanyBalanceRepo: companyBalanceRepo,
	}
}

// getProbabilitiesBasedOnBalance - Определение диапазонов баланса и соответствующих вероятностей
func (service *SlotGameService) getProbabilitiesBasedOnBalance(balance float64) (jackpotProb, tripleMatchProb, doubleSevenProb, singleSevenProb, noWinProb float64) {
	var jackpot, tripleMatch, doubleSeven, singleSeven, noWin float64

	// Влияние баланса на вероятность
	switch {
	case balance <= 1:
		noWin = 1

	case balance <= 50:
		// Низкие шансы на выигрыш (1-50 тонн)
		jackpot = 0.002
		tripleMatch = 0.01
		doubleSeven = 0.02
		singleSeven = 0.05
		noWin = 0.918

	case balance <= 100:
		// Средние шансы на выигрыш (51-100 тонн)
		jackpot = 0.005
		tripleMatch = 0.02
		doubleSeven = 0.05
		singleSeven = 0.10
		noWin = 0.804

	case balance >= 101:
		// Высокие шансы на выигрыш (101 тонн и более)
		jackpot = 0.01
		tripleMatch = 0.03
		doubleSeven = 0.07
		singleSeven = 0.15
		noWin = 0.74
	}

	return jackpot, tripleMatch, doubleSeven, singleSeven, noWin
}

// generateCombinationBasedOnBalance - Генерация комбинации с учётом вероятности в зависимости от баланса
func (service *SlotGameService) generateCombinationBasedOnBalance(balance float64) []int {
	// Получаем вероятности на основе баланса
	jackpotProb, tripleMatchProb, doubleSevenProb, singleSevenProb, _ := service.getProbabilitiesBasedOnBalance(balance)

	// Генерируем случайное число от 0 до 1
	randVal := rand.Float64()

	// Определяем, какая комбинация выпадет в зависимости от вероятности
	switch {
	case randVal < jackpotProb:
		// Выпадает Jackpot
		return []int{7, 7, 7}
	case randVal < jackpotProb+tripleMatchProb:
		// Выпадение трёх одинаковых чисел (не "777")
		num := rand.Intn(6) + 1
		return []int{num, num, num}
	case randVal < jackpotProb+tripleMatchProb+doubleSevenProb:
		// Выпадение двух семёрок
		return service.generateDoubleSevenCombination()
	case randVal < jackpotProb+tripleMatchProb+doubleSevenProb+singleSevenProb:
		// Выпадение одной семёрки
		combination := []int{0, 0, 0}

		// Генерируем случайную позицию для семёрки
		sevenPos := rand.Intn(3)
		combination[sevenPos] = 7

		// Заполняем остальные две позиции случайными числами (не семёрками)
		for i := 0; i < 3; i++ {
			if i != sevenPos {
				combination[i] = rand.Intn(6) + 1 // Числа от 1 до 6
			}
		}

		return combination
	default:
		// NoWin: все остальные комбинации
		return service.generateLosingCombination()
	}
}

// generateDoubleSevenCombination - Метод для генерации комбинации с двумя семёрками (x5)
func (service *SlotGameService) generateDoubleSevenCombination() []int {
	// Все возможные комбинации с двумя семёрками
	combinations := [][]int{
		{7, 7, 1}, {7, 7, 2}, {7, 7, 3}, {7, 7, 4}, {7, 7, 5}, {7, 7, 6},
		{7, 1, 7}, {7, 2, 7}, {7, 3, 7}, {7, 4, 7}, {7, 5, 7}, {7, 6, 7},
		{1, 7, 7}, {2, 7, 7}, {3, 7, 7}, {4, 7, 7}, {5, 7, 7}, {6, 7, 7},
	}

	// Выбираем случайную комбинацию
	combination := combinations[rand.Intn(len(combinations))]

	// Перемешиваем элементы комбинации для большей случайности
	rand.Shuffle(len(combination), func(i, j int) {
		combination[i], combination[j] = combination[j], combination[i]
	})

	return combination
}

// PlaySlot - Основной метод для игры в слоты
func (service *SlotGameService) PlaySlot(ctx context.Context, wallet string, ton float64, cubes int) ([]int, float64, error) {
	// Проверяем корректность ставки: либо ton > 0, либо cubes > 0, но не оба и не оба равны нулю
	if (ton > 0 && cubes > 0) || (ton == 0 && cubes == 0) {
		return nil, 0, fmt.Errorf("invalid bet: specify either ton or cubes, but not both")
	}

	// Дополнительная валидация для ставок в тонах
	if ton > 0 {
		if ton < MinTonBet || ton > MaxTonBet {
			return nil, 0, fmt.Errorf("invalid ton bet: minimum bet is %.1f ton and maximum bet is %.1f ton", MinTonBet, MaxTonBet)
		}
	}

	// Дополнительная валидация для ставок в кубах
	if cubes > 0 {
		if cubes < MinCubeBet || cubes > MaxCubeBet {
			return nil, 0, fmt.Errorf("invalid cube bet: minimum bet is %d cube and maximum bet is %d cubes", MinCubeBet, MaxCubeBet)
		}
	}

	// Получаем баланс пользователя
	balanceData, err := service.UserRepo.GetUserBalances(ctx, wallet)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve user balance: %v", err)
	}

	// Извлекаем баланс кубов
	cubeBalance := 0
	if cubes > 0 {
		cubeVal, exists := balanceData["cubes"]
		if !exists {
			return nil, 0, fmt.Errorf("not enough cubes for the bet")
		}

		switch v := cubeVal.(type) {
		case int:
			cubeBalance = v
		case int64:
			cubeBalance = int(v)
		case float64:
			cubeBalance = int(v)
		default:
			return nil, 0, fmt.Errorf("invalid cube balance format")
		}

		if cubeBalance < cubes {
			return nil, 0, fmt.Errorf("not enough cubes for the bet")
		}
	}

	// Извлекаем баланс тонн
	tonBalance := 0.0
	if ton > 0 {
		tonVal, exists := balanceData["ton_balance"]
		if !exists {
			return nil, 0, fmt.Errorf("not enough tons for the bet")
		}

		switch v := tonVal.(type) {
		case float64:
			tonBalance = v
		case int:
			tonBalance = float64(v)
		default:
			return nil, 0, fmt.Errorf("invalid ton balance format")
		}

		if tonBalance < ton {
			return nil, 0, fmt.Errorf("not enough tons for the bet")
		}
	}

	// Списываем ставку
	if ton > 0 {
		err := service.UserRepo.AddTokens(ctx, wallet, map[string]float64{"ton_balance": -ton})
		if err != nil {
			return nil, 0, fmt.Errorf("failed to deduct ton balance: %v", err)
		}
	}
	if cubes > 0 {
		err := service.UserRepo.AddCubes(ctx, wallet, -cubes)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to deduct cube balance: %v", err)
		}
	}

	// Получаем баланс слотов (компании)
	slotBalance, err := service.CompanyBalanceRepo.GetBalance(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve slot balance: %v", err)
	}
	if slotBalance == nil {
		return nil, 0, fmt.Errorf("slot balance is not initialized")
	}

	// Определение суммы ставки
	var bet float64
	if ton > 0 {
		bet = ton
	} else {
		bet = float64(cubes) * CubeToTonRate
	}

	// Генерируем комбинацию
	var combination []int
	if slotBalance.Tons < bet*10 {
		combination = service.generateLosingCombination() // Гарантированный проигрыш
	} else {
		combination = service.generateCombinationBasedOnBalance(slotBalance.Tons) // Обычная генерация
	}

	// Подсчёт одинаковых чисел
	countMap := make(map[int]int)
	for _, num := range combination {
		countMap[num]++
	}

	// Логика определения выигрыша
	var winnings float64
	switch {
	case countMap[7] == 3:
		winnings = bet * 10 // Джекпот
	case countMap[7] == 2:
		winnings = bet * 5 // Две семёрки
	case countMap[7] == 1 && hasPairApartFromSeven(countMap):
		winnings = bet * 5 // Одна семёрка и пара других чисел
	case countMap[7] == 1:
		winnings = bet * 2 // Одна семёрка
	case hasThreeOfAKind(countMap):
		winnings = bet * 3 // Три одинаковых числа
	default:
		winnings = 0 // Проигрыш
	}

	// Если проигрыш, добавляем ставку к балансу компании
	if winnings == 0 {
		err := service.CompanyBalanceRepo.AddTokens(ctx, "tons", bet)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to add bet to slot balance: %v", err)
		}
	}

	// Если выигрыш есть, начисляем его
	if winnings > 0 {
		err := service.addTonWinnings(ctx, wallet, winnings+bet)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to add ton winnings: %v", err)
		}
	}

	// Определяем тип ставки
	betType := "ton"
	if cubes > 0 {
		betType = "cubes"
	}

	// Преобразуем комбинацию в строку
	resultStr := fmt.Sprintf("%v", combination)

	// Записываем игру
	err = service.SlotRepository.RecordGame(ctx, wallet, bet, betType, resultStr, winnings)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to record game: %v", err)
	}

	return combination, winnings, nil
}

// generateLosingCombination - Метод для генерации гарантированно проигрышной комбинации
func (service *SlotGameService) generateLosingCombination() []int {
	// Базовый список проигрышных комбинаций, которые не дают выигрышных результатов
	losingCombinations := [][]int{
		{1, 2, 3}, {2, 3, 4}, {4, 5, 6}, {1, 3, 5}, {2, 4, 6},
		{3, 4, 5}, {6, 2, 3}, {1, 3, 4}, {4, 1, 6}, {6, 3, 5}, {2, 5, 6},
		{6, 5, 4},
		{3, 4, 1}, {6, 2, 3}, {1, 6, 4}, {4, 1, 4}, {1, 3, 5}, {1, 5, 1},
		{3, 1, 1}, {1, 6, 5}, {2, 1, 4}, {6, 6, 4}, {2, 3, 5}, {3, 5, 1},
		{2, 1, 4},
		{3, 3, 5},
		{4, 2, 6},
		{2, 4, 1},
		{3, 4, 1},
		{6, 2, 4},
		{1, 2, 4},
		{1, 3, 5},
		{4, 1, 6},
		{5, 2, 3}, {4, 3, 6}, {2, 3, 6}, {1, 5, 6}, {2, 5, 4},
		{6, 4, 2}, {3, 2, 5}, {1, 6, 3}, {4, 5, 1}, {2, 6, 1},
		{5, 3, 2}, {6, 1, 4}, {3, 5, 6}, {2, 4, 5}, {1, 2, 6},
		{5, 6, 3}, {3, 1, 4}, {4, 2, 5}, {6, 3, 2}, {1, 4, 5},
		{2, 3, 1}, {5, 1, 6}, {4, 6, 2}, {3, 2, 4}, {1, 5, 3},
		{6, 4, 5}, {5, 3, 4}, {2, 1, 6}, {4, 3, 5}, {1, 6, 2},
		{3, 2, 6}, {5, 4, 2}, {6, 1, 3}, {4, 6, 1}, {2, 5, 3},
		{1, 3, 6}, {3, 4, 6}, {5, 2, 4}, {6, 5, 3}, {4, 1, 3},
		{2, 3, 4}, {6, 2, 5}, {1, 4, 6}, {3, 5, 2}, {5, 6, 1},
		{4, 3, 2}, {6, 4, 3}, {2, 5, 1}, {3, 1, 6}, {1, 2, 5},
		{5, 3, 6}, {6, 2, 1}, {4, 6, 5}, {2, 3, 5}, {1, 4, 3},
		{5, 2, 6}, {4, 1, 5}, {6, 3, 4}, {3, 2, 1}, {1, 5, 4},
		{2, 4, 6}, {6, 1, 5}, {3, 4, 2}, {4, 6, 3}, {5, 1, 3},
		{2, 6, 5}, {1, 3, 4}, {5, 4, 6}, {6, 2, 3}, {3, 5, 4},
		{4, 1, 2}, {6, 5, 1}, {2, 3, 6}, {1, 4, 2}, {5, 3, 1},
		{6, 4, 5}, {3, 2, 6}, {4, 5, 3}, {2, 1, 5}, {1, 6, 4},
		{5, 2, 1}, {4, 3, 1}, {6, 1, 2}, {3, 4, 5}, {2, 5, 6},
	}

	// Выбираем случайную комбинацию из списка
	combination := losingCombinations[rand.Intn(len(losingCombinations))]

	// Перемешиваем элементы комбинации для большей случайности
	rand.Shuffle(len(combination), func(i, j int) {
		combination[i], combination[j] = combination[j], combination[i]
	})

	return combination
}

// hasThreeOfAKind - Проверка на три одинаковых числа (любых, кроме семёрок)
func hasThreeOfAKind(countMap map[int]int) bool {
	for num, count := range countMap {
		if count == 3 && num != 7 {
			return true
		}
	}
	return false
}

// hasPairApartFromSeven - Проверка наличия пары одинаковых чисел, отличных от семёрок
func hasPairApartFromSeven(countMap map[int]int) bool {
	for num, count := range countMap {
		if num != 7 && count == 2 {
			return true
		}
	}
	return false
}

// addTonWinnings - Добавление выигрыша в тонах пользователю и списание с баланса слотов.
func (service *SlotGameService) addTonWinnings(ctx context.Context, wallet string, winnings float64) error {
	// Добавляем тоны пользователю
	err := service.UserRepo.AddTokens(ctx, wallet, map[string]float64{"ton_balance": winnings})
	if err != nil {
		return fmt.Errorf("failed to add ton winnings to user: %v", err)
	}

	// Списываем тоны с баланса компании
	err = service.CompanyBalanceRepo.DeductTons(ctx, winnings)
	if err != nil {
		return fmt.Errorf("failed to deduct ton winnings from company balance: %v", err)
	}

	return nil
}

// RecordGame - Метод для записи игры
func (service *SlotGameService) RecordGame(ctx context.Context, wallet string, bet float64, betType string, result string, winAmount float64) error {
	// Используем метод репозитория для записи игры.
	return service.SlotRepository.RecordGame(ctx, wallet, bet, betType, result, winAmount)
}

// GetGamesByWallet - Получить все игры игрока по кошельку.
func (service *SlotGameService) GetGamesByWallet(ctx context.Context, wallet string, limit int64) ([]slotEntities.SlotGame, error) {
	// Используем метод репозитория для получения игр по кошельку.
	return service.SlotRepository.GetGamesByWallet(ctx, wallet, limit)
}

// GetRecentGames - Получить последние игры по кошельку.
func (service *SlotGameService) GetRecentGames(ctx context.Context, wallet string, limit int64) ([]slotEntities.SlotGame, error) {
	// Используем метод репозитория для получения последних игр.
	return service.SlotRepository.GetRecentGames(ctx, wallet, limit)
}
