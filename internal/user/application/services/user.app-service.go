package services

import (
	"context"

	"github.com/Peranum/tg-dice/internal/user/domain/entities"
	"github.com/Peranum/tg-dice/internal/user/domain/services"
	"github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
)

type UserAppService struct {
	DomainService     *services.UserDomainService
	WithdrawalService *services.WithdrawalService
}

func NewUserAppService(domainService *services.UserDomainService, withdrawalService *services.WithdrawalService) *UserAppService {
	return &UserAppService{
		DomainService:     domainService,
		WithdrawalService: withdrawalService,
	}
}

func (as *UserAppService) CreateUser(ctx context.Context, user *entities.User) (*entities.User, error) {
	return as.DomainService.CreateUser(ctx, user)
}

func (as *UserAppService) GetUser(ctx context.Context, id string) (*entities.User, error) {
	return as.DomainService.GetUserByID(ctx, id)
}

func (as *UserAppService) GetUserByWallet(ctx context.Context, wallet string) (*entities.User, error) {
	return as.DomainService.GetUserByWallet(ctx, wallet)
}

func (as *UserAppService) PatchUserByTgID(ctx context.Context, tgid string, updateData map[string]interface{}) error {
	return as.DomainService.PatchUserByTgID(ctx, tgid, updateData)
}

func (as *UserAppService) DeleteUser(ctx context.Context, id string) error {
	return as.DomainService.DeleteUser(ctx, id)
}

func (as *UserAppService) ListUsers(ctx context.Context, limit int64, offset int64) ([]entities.User, error) {
	return as.DomainService.ListUsers(ctx, limit, offset)
}

func (as *UserAppService) GetTokenBalance(ctx context.Context, wallet string, tokenType string) (float64, error) {
	return as.DomainService.GetTokenBalance(ctx, wallet, tokenType)
}

func (as *UserAppService) GetUserBalances(ctx context.Context, wallet string) (map[string]interface{}, error) {
	return as.DomainService.GetUserBalances(ctx, wallet)
}

func (as *UserAppService) GetReferralCodeByWallet(ctx context.Context, wallet string) (string, error) {
	return as.DomainService.GetReferralCodeByWallet(ctx, wallet)
}

func (as *UserAppService) GetReferralEarnings(ctx context.Context, wallet string) (map[string]float64, error) {
	return as.DomainService.GetUserReferralEarnings(ctx, wallet)
}

func (as *UserAppService) GetUserByName(ctx context.Context, name string) (*entities.User, error) {
	return as.DomainService.GetUserByName(ctx, name)
}

func (as *UserAppService) GetUserPointsByWallet(ctx context.Context, wallet string) (float64, error) {
	return as.DomainService.GetUserPointsByWallet(ctx, wallet)
}

func (as *UserAppService) GetUsersSortedByPoints(ctx context.Context, limit int64, offset int64) ([]entities.User, error) {
	return as.DomainService.GetUsersSortedByPoints(ctx, limit, offset)
}

// Методы WithdrawalService
func (as *UserAppService) CreateWithdrawal(ctx context.Context, amount float64, wallet string, jettonName *string) error {
	return as.WithdrawalService.CreateWithdrawal(ctx, amount, wallet, jettonName)
}

func (as *UserAppService) GetWithdrawal(ctx context.Context, id string) (*repositories.Withdrawal, error) {
	return as.WithdrawalService.GetWithdrawal(ctx, id)
}

func (as *UserAppService) GetWithdrawalsByWallet(ctx context.Context, wallet string, limit int64) ([]repositories.Withdrawal, error) {
	return as.WithdrawalService.GetWithdrawalsByWallet(ctx, wallet, limit)
}

func (as *UserAppService) GetLast50Withdrawals(ctx context.Context) ([]repositories.Withdrawal, error) {
	return as.WithdrawalService.GetLast50Withdrawals(ctx)
}

func (as *UserAppService) DeleteWithdrawal(ctx context.Context, id string) error {
	return as.WithdrawalService.DeleteWithdrawal(ctx, id)
}

// GetLast50WithdrawalsWithJetton retrieves the last 50 withdrawals with a specific JettonName.
func (as *UserAppService) GetLast50WithdrawalsWithJetton(ctx context.Context, jettonName string) ([]repositories.Withdrawal, error) {
	return as.WithdrawalService.GetLast50WithdrawalsWithJetton(ctx, jettonName)
}

// GetLast50WithdrawalsWithoutJetton retrieves the last 50 withdrawals where the jetton_name field is missing or does not exist.
func (as *UserAppService) GetLast50WithdrawalsWithoutJetton(ctx context.Context) ([]repositories.Withdrawal, error) {
	return as.WithdrawalService.GetLast50WithdrawalsWithoutJetton(ctx)
}
