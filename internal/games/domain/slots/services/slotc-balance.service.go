package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Peranum/tg-dice/internal/games/infrastructure/slots/entities"
	"github.com/Peranum/tg-dice/internal/games/infrastructure/slots/repositories"
)

// SlotsBalanceService - Сервис для работы с балансом.
type SlotsBalanceService struct {
	repo *repositories.SlotsBalanceRepository
}

// NewSlotsBalanceService - Создает новый сервис для работы с балансом.
func NewSlotsBalanceService(repo *repositories.SlotsBalanceRepository) *SlotsBalanceService {
	return &SlotsBalanceService{
		repo: repo,
	}
}

// InitializeBalance - Инициализация общего баланса.
func (s *SlotsBalanceService) InitializeBalance(ctx context.Context, tons, cubes float64) error {
	if tons < 0 || cubes < 0 {
		return fmt.Errorf("tons and cubes must be non-negative")
	}
	return s.repo.InitializeBalance(ctx, tons, cubes)
}

// GetBalance - Получить текущий баланс.
func (s *SlotsBalanceService) GetBalance(ctx context.Context) (*entities.SlotsBalance, error) {
	balance, err := s.repo.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %v", err)
	}
	if balance == nil {
		return &entities.SlotsBalance{
			Tons:      0,
			Cubes:     0,
			UpdatedAt: time.Now(),
		}, nil
	}
	return balance, nil
}

// UpdateBalance - Обновить баланс.
func (s *SlotsBalanceService) UpdateBalance(ctx context.Context, tonsDelta, cubesDelta float64) error {
	return s.repo.UpdateBalance(ctx, tonsDelta, cubesDelta)
}

func (s *SlotsBalanceService) AddTokens(ctx context.Context, tokenType string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("сумма добавляемых токенов должна быть положительной")
	}

	if tokenType != "tons" && tokenType != "cubes" {
		return fmt.Errorf("неизвестный тип токенов: %s", tokenType)
	}

	err := s.repo.AddTokens(ctx, tokenType, amount)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении токенов: %v", err)
	}

	return nil
}

// SubtractTokens - Вычитает токены указанного типа из баланса, проверяя достаточность.
func (s *SlotsBalanceService) SubtractTokens(ctx context.Context, tokenType string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("сумма вычитаемых токенов должна быть положительной")
	}

	if tokenType != "tons" && tokenType != "cubes" {
		return fmt.Errorf("неизвестный тип токенов: %s", tokenType)
	}

	err := s.repo.SubtractTokens(ctx, tokenType, amount)
	if err != nil {
		return fmt.Errorf("ошибка при вычитании токенов: %v", err)
	}

	return nil
}
