package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Peranum/tg-dice/internal/promocodes/infrastructure/entity"
	"github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// PromoCodeRepository handles database operations for promocodes
type PromoCodeRepository struct {
	UserRepo   *repositories.UserRepository
	Collection *mongo.Collection
}

// NewPromoCodeRepository creates a new PromoCodeRepository
func NewPromoCodeRepository(db *mongo.Database, userRepo *repositories.UserRepository) *PromoCodeRepository {
	return &PromoCodeRepository{
		UserRepo:   userRepo,
		Collection: db.Collection("promocodes"),
	}
}

// CreatePromoCode creates a new promocode
func (r *PromoCodeRepository) CreatePromoCode(ctx context.Context, promo *entity.PromoCodeEntity) error {
	promo.UsedActivations = 0
	promo.Status = entity.Active
	promo.CreatedAt = time.Now()
	promo.UpdatedAt = time.Now()

	// Initialize the activated_wallets field as an empty array
	if promo.ActivatedWallets == nil {
		promo.ActivatedWallets = []string{}
	}

	_, err := r.Collection.InsertOne(ctx, promo)
	if err != nil {
		log.Printf("[CreatePromoCode] Error creating promocode: %v", err)
		return err
	}
	log.Printf("[CreatePromoCode] Promocode created successfully: %s", promo.Code)
	return nil
}

// GetPromoCodeByCode retrieves a promocode by its code
func (r *PromoCodeRepository) GetPromoCodeByCode(ctx context.Context, code string) (*entity.PromoCodeEntity, error) {
	var promo entity.PromoCodeEntity
	err := r.Collection.FindOne(ctx, bson.M{"code": code}).Decode(&promo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[GetPromoCodeByCode] No promocode found for code: %s", code)
			return nil, errors.New("promocode not found")
		}
		log.Printf("[GetPromoCodeByCode] Error fetching promocode: %v", err)
		return nil, err
	}
	return &promo, nil
}

// ListActivePromoCodes retrieves all active promocodes
func (r *PromoCodeRepository) ListActivePromoCodes(ctx context.Context) ([]entity.PromoCodeEntity, error) {
	filter := bson.M{"status": entity.Active}
	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		log.Printf("[ListActivePromoCodes] Error fetching active promocodes: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var promocodes []entity.PromoCodeEntity
	if err := cursor.All(ctx, &promocodes); err != nil {
		log.Printf("[ListActivePromoCodes] Error decoding promocodes: %v", err)
		return nil, err
	}
	return promocodes, nil
}

// UpdatePromoCodeStatus updates the status of a promocode by its code
func (r *PromoCodeRepository) UpdatePromoCodeStatus(ctx context.Context, code string, status entity.PromoCodeStatus) error {
	result, err := r.Collection.UpdateOne(
		ctx,
		bson.M{"code": code},
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}},
	)
	if err != nil {
		log.Printf("[UpdatePromoCodeStatus] Error updating promocode status: %v", err)
		return err
	}
	if result.MatchedCount == 0 {
		log.Printf("[UpdatePromoCodeStatus] No promocode found with code: %s", code)
		return errors.New("promocode not found")
	}
	return nil
}

// IncrementUsedActivations increments the `used_activations` count for a promocode by its code
func (r *PromoCodeRepository) IncrementUsedActivations(ctx context.Context, code string) error {
	result, err := r.Collection.UpdateOne(
		ctx,
		bson.M{"code": code, "status": entity.Active},
		bson.M{
			"$inc": bson.M{"used_activations": 1},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		log.Printf("[IncrementUsedActivations] Error incrementing activations: %v", err)
		return err
	}
	if result.MatchedCount == 0 {
		log.Printf("[IncrementUsedActivations] No active promocode found with code: %s", code)
		return errors.New("promocode not found or not active")
	}
	return nil
}

// ListPromoCodesByStatus retrieves all promocodes by their status
func (r *PromoCodeRepository) ListPromoCodesByStatus(ctx context.Context, status entity.PromoCodeStatus) ([]entity.PromoCodeEntity, error) {
	filter := bson.M{"status": status}
	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		log.Printf("[ListPromoCodesByStatus] Error fetching promocodes by status: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var promocodes []entity.PromoCodeEntity
	if err := cursor.All(ctx, &promocodes); err != nil {
		log.Printf("[ListPromoCodesByStatus] Error decoding promocodes: %v", err)
		return nil, err
	}
	return promocodes, nil
}

// ExpirePromocodes expires all promocodes that have passed their expiration date
func (r *PromoCodeRepository) ExpirePromocodes(ctx context.Context) error {
	now := time.Now()
	result, err := r.Collection.UpdateMany(
		ctx,
		bson.M{"expires_at": bson.M{"$lt": now}, "status": entity.Active},
		bson.M{"$set": bson.M{"status": entity.Expired, "updated_at": now}},
	)
	if err != nil {
		log.Printf("[ExpirePromocodes] Error expiring promocodes: %v", err)
		return err
	}
	log.Printf("[ExpirePromocodes] Expired %d promocodes", result.ModifiedCount)
	return nil
}

func (r *PromoCodeRepository) ActivatePromoCode(ctx context.Context, wallet string, code string, userRepo *repositories.UserRepository) error {
	log.Printf("[ActivatePromoCode] Activating promocode: %s for wallet: %s", code, wallet)

	// Retrieve the promocode
	var promo entity.PromoCodeEntity
	err := r.Collection.FindOne(ctx, bson.M{
		"code":   code,
		"status": entity.Active,
	}).Decode(&promo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[ActivatePromoCode] Promocode not found or inactive: %s", code)
			return errors.New("promocode not found or inactive")
		}
		log.Printf("[ActivatePromoCode] Error fetching promocode: %v", err)
		return err
	}

	// Check if wallet has already activated the promocode
	if contains(promo.ActivatedWallets, wallet) {
		log.Printf("[ActivatePromoCode] Wallet has already activated promocode: %s", wallet)
		return errors.New("promocode already activated by this wallet")
	}

	// Check if promocode activations are exhausted
	if promo.UsedActivations >= promo.MaxActivations {
		log.Printf("[ActivatePromoCode] Promocode activations exhausted: %s", code)
		return errors.New("promocode activations exhausted")
	}

	// Apply rewards
	err = userRepo.ApplyPromoCodeRewards(ctx, wallet, promo.TokenType, promo.Amount)
	if err != nil {
		log.Printf("[ActivatePromoCode] Error applying rewards: %v", err)
		return err
	}

	// Increment activations and update wallet list
	_, err = r.Collection.UpdateOne(
		ctx,
		bson.M{"code": code},
		bson.M{
			"$inc":      bson.M{"used_activations": 1},
			"$addToSet": bson.M{"activated_wallets": wallet},
			"$set":      bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		log.Printf("[ActivatePromoCode] Error updating promocode: %v", err)
		return err
	}

	log.Printf("[ActivatePromoCode] Promocode activated successfully for wallet: %s", wallet)
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
