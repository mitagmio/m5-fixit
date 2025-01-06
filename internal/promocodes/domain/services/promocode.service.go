package services

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Peranum/tg-dice/internal/promocodes/infrastructure/entity"
	"github.com/Peranum/tg-dice/internal/promocodes/infrastructure/repository"
	"github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
)

type PromoCodeService struct {
	promoRepo *repository.PromoCodeRepository
	userRepo  *repositories.UserRepository
}

// NewPromoCodeService creates a new PromoCodeService instance
func NewPromoCodeService(promoRepo *repository.PromoCodeRepository, userRepo *repositories.UserRepository) *PromoCodeService {
	return &PromoCodeService{
		promoRepo: promoRepo,
		userRepo:  userRepo,
	}
}

// CreatePromoCode handles creating a new promocode
func (s *PromoCodeService) CreatePromoCode(ctx context.Context, promo *entity.PromoCodeEntity) error {
	log.Printf("[PromoCodeService] Creating promocode: %+v", promo)

	// Validate promocode fields
	if promo.Code == "" || len(promo.Code) < 3 {
		return errors.New("promocode must have at least 3 characters")
	}
	if promo.Amount <= 0 {
		return errors.New("reward amount must be greater than 0")
	}
	if promo.MaxActivations <= 0 {
		return errors.New("max activations must be greater than 0")
	}
	if promo.ExpiresAt != nil && promo.ExpiresAt.Before(time.Now()) {
		return errors.New("expiration date must be in the future")
	}

	// Delegate creation to the repository
	err := s.promoRepo.CreatePromoCode(ctx, promo)
	if err != nil {
		log.Printf("[PromoCodeService] Error creating promocode: %v", err)
		return err
	}

	log.Printf("[PromoCodeService] Promocode created successfully: %s", promo.Code)
	return nil
}

// ActivatePromoCode handles promocode activation and rewards application
func (s *PromoCodeService) ActivatePromoCode(ctx context.Context, wallet string, code string) error {
	log.Printf("[PromoCodeService] Activating promocode: %s for wallet: %s", code, wallet)

	// Check if the user exists
	userExists, err := s.userRepo.DoesUserExist(ctx, wallet)
	if err != nil {
		log.Printf("[PromoCodeService] Error checking user existence: %v", err)
		return errors.New("failed to verify user existence")
	}
	if !userExists {
		log.Printf("[PromoCodeService] User not found for wallet: %s", wallet)
		return errors.New("user not found")
	}

	// Activate the promocode
	err = s.promoRepo.ActivatePromoCode(ctx, wallet, code, s.userRepo)
	if err != nil {
		log.Printf("[PromoCodeService] Error activating promocode: %v", err)
		return err
	}

	log.Printf("[PromoCodeService] Promocode activated successfully for wallet: %s", wallet)
	return nil
}

// ListActivePromoCodes retrieves all active promocodes
func (s *PromoCodeService) ListActivePromoCodes(ctx context.Context) ([]entity.PromoCodeEntity, error) {
	log.Printf("[PromoCodeService] Listing active promocodes")

	promocodes, err := s.promoRepo.ListActivePromoCodes(ctx)
	if err != nil {
		log.Printf("[PromoCodeService] Error listing active promocodes: %v", err)
		return nil, err
	}

	return promocodes, nil
}

// GetPromoCodeByCode retrieves a promocode by its code
func (s *PromoCodeService) GetPromoCodeByCode(ctx context.Context, code string) (*entity.PromoCodeEntity, error) {
	log.Printf("[PromoCodeService] Retrieving promocode by code: %s", code)

	promo, err := s.promoRepo.GetPromoCodeByCode(ctx, code)
	if err != nil {
		log.Printf("[PromoCodeService] Error retrieving promocode: %v", err)
		return nil, err
	}

	return promo, nil
}

// ExpirePromocodes marks expired promocodes as expired
func (s *PromoCodeService) ExpirePromocodes(ctx context.Context) error {
	log.Printf("[PromoCodeService] Expiring old promocodes")

	err := s.promoRepo.ExpirePromocodes(ctx)
	if err != nil {
		log.Printf("[PromoCodeService] Error expiring promocodes: %v", err)
		return err
	}

	log.Printf("[PromoCodeService] Expired old promocodes successfully")
	return nil
}
