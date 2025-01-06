package mapper

import (
	"errors"
	"github.com/Peranum/tg-dice/internal/user/domain/entities"
	"github.com/Peranum/tg-dice/internal/user/infrastructure/odm-entities"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ToDomain(odmEntity *odm_entities.UserEntity) *entities.User {
	return &entities.User{
		ID:               odmEntity.ID.Hex(),
		Name:             odmEntity.Name,
		FirstName:        odmEntity.FirstName, // Добавлено поле FirstName
		Wallet:           odmEntity.Wallet,
		Ton_balance:      odmEntity.Ton_balance,
		M5_balance:       odmEntity.M5_balance,
		Dfc_balance:      odmEntity.Dfc_balance,
		Cubes:            odmEntity.Cubes,
		ReferralCode:     odmEntity.ReferralCode,
		ReferredBy:       odmEntity.ReferredBy,
		ReferralEarnings: odmEntity.ReferralEarnings,
		Points:           odmEntity.Points,
		TgID:             odmEntity.TgID,
		Language:         odmEntity.Language,
		CreatedAt:        odmEntity.CreatedAt,
		UpdatedAt:        odmEntity.UpdatedAt,
	}
}

func ToODM(domainEntity *entities.User) (*odm_entities.UserEntity, error) {
	var objectID primitive.ObjectID
	var err error

	if domainEntity.ID != "" {
		objectID, err = primitive.ObjectIDFromHex(domainEntity.ID)
		if err != nil {
			return nil, errors.New("invalid ObjectID format")
		}
	} else {
		objectID = primitive.NewObjectID()
	}

	if domainEntity.ReferralCode == "" {
		return nil, errors.New("referral code cannot be empty")
	}

	if domainEntity.Language != "ru" && domainEntity.Language != "eng" {
		return nil, errors.New("language must be either RU or ENG")
	}

	return &odm_entities.UserEntity{
		ID:               objectID,
		Name:             domainEntity.Name,
		FirstName:        domainEntity.FirstName, // Добавлено поле FirstName
		Wallet:           domainEntity.Wallet,
		Ton_balance:      domainEntity.Ton_balance,
		M5_balance:       domainEntity.M5_balance,
		Dfc_balance:      domainEntity.Dfc_balance,
		Cubes:            domainEntity.Cubes,
		ReferralCode:     domainEntity.ReferralCode,
		ReferredBy:       domainEntity.ReferredBy,
		ReferralEarnings: domainEntity.ReferralEarnings,
		Points:           domainEntity.Points,
		TgID:             domainEntity.TgID,
		Language:         domainEntity.Language,
		CreatedAt:        domainEntity.CreatedAt,
		UpdatedAt:        domainEntity.UpdatedAt,
	}, nil
}
