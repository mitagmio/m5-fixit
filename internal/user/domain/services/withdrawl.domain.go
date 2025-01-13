package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
)

type WithdrawalService struct {
	Repo     *repositories.WithdrawalsRepository
	UserRepo *repositories.UserRepository // Add this field
}

// NewWithdrawalService creates a new instance of WithdrawalService.
// NewWithdrawalService creates a new instance of WithdrawalService.
func NewWithdrawalService(repo *repositories.WithdrawalsRepository, userRepo *repositories.UserRepository) *WithdrawalService {
	return &WithdrawalService{
		Repo:     repo,
		UserRepo: userRepo, // Initialize UserRepo
	}
}

func (s *WithdrawalService) CreateWithdrawal(ctx context.Context, amount float64, wallet string, jettonName *string) error {
	if amount <= 0 {
		return errors.New("withdrawal amount must be greater than zero")
	}

	// Default to ton_balance if jettonName is not provided
	if jettonName == nil {
		jettonName = nil // Don't assign a default value, leave it nil
	}

	// Map the jettonName to the corresponding token type if it's provided
	var tokenType string
	if jettonName != nil {
		switch *jettonName {
		case "m5":
			tokenType = "m5_balance"
		case "dfc":
			tokenType = "dfc_balance"
		case "ton":
			tokenType = "ton_balance"
		default:
			tokenType = "ton_balance" // Default to ton_balance if jettonName is provided but not recognized
		}
	} else {
		tokenType = "ton_balance" // Default to ton_balance if jettonName is not provided
	}

	// maxLimits := map[string]float64{
	// 	"ton_balance": 10.0,
	// 	"m5_balance":  10.0,
	// 	"dfc_balance": 10.0,
	// }

	// // Check if the requested amount exceeds the maximum limit
	// if limit, exists := maxLimits[tokenType]; exists {
	// 	if amount > limit {
	// 		return fmt.Errorf("withdrawal amount exceeds the maximum limit of %.2f for %s", limit, tokenType)
	// 	}
	// } else {
	// 	return errors.New("invalid token type")
	// }

	// Check if the user has sufficient balance for the withdrawal
	hasSufficientBalance, err := s.UserRepo.HasSufficientBalance(ctx, wallet, tokenType, amount)
	if err != nil {
		return fmt.Errorf("error checking balance: %v", err)
	}

	if !hasSufficientBalance {
		return errors.New("insufficient balance")
	}

	// Create the withdrawal record
	withdrawal := repositories.NewWithdrawal(wallet, amount)
	if jettonName != nil {
		withdrawal.JettonName = *jettonName
	} else {
		withdrawal.JettonName = "ton"
	}

	// Create the withdrawal record
	return s.Repo.CreateWithdrawal(ctx, withdrawal)
}

// GetWithdrawal retrieves a withdrawal by its ID.
func (s *WithdrawalService) GetWithdrawal(ctx context.Context, id string) (*repositories.Withdrawal, error) {
	return s.Repo.GetWithdrawalByID(ctx, id)
}

// GetWithdrawalsByWallet retrieves withdrawals by wallet with an optional limit.
func (s *WithdrawalService) GetWithdrawalsByWallet(ctx context.Context, wallet string, limit int64) ([]repositories.Withdrawal, error) {
	return s.Repo.GetWithdrawalsByWallet(ctx, wallet, limit)
}

// GetLast50Withdrawals retrieves the last 50 withdrawals.
func (s *WithdrawalService) GetLast50Withdrawals(ctx context.Context) ([]repositories.Withdrawal, error) {
	return s.Repo.GetLast50Withdrawals(ctx)
}

// DeleteWithdrawal handles the deletion of a withdrawal by its ID.
func (s *WithdrawalService) DeleteWithdrawal(ctx context.Context, id string) error {
	return s.Repo.DeleteWithdrawal(ctx, id)
}

// GetLast50WithdrawalsWithJetton retrieves the last 50 withdrawals,
// optionally filtering by a specific JettonName.
func (s *WithdrawalService) GetLast50WithdrawalsWithJetton(ctx context.Context, jettonName string) ([]repositories.Withdrawal, error) {
	return s.Repo.GetLast50WithdrawalsWithJetton(ctx, jettonName)
}

// GetLast50WithdrawalsWithoutJetton retrieves the last 50 withdrawals
// where the jetton_name field is missing or does not exist.
func (s *WithdrawalService) GetLast50WithdrawalsWithoutJetton(ctx context.Context) ([]repositories.Withdrawal, error) {
	return s.Repo.GetLast50WithdrawalsWithoutJetton(ctx)
}
