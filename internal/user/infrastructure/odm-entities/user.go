package odm_entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserEntity struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name             string             `bson:"name" json:"name"`
	FirstName        string             `bson:"first_name" json:"first_name"` // Новое поле FirstName
	Wallet           string             `bson:"wallet" json:"wallet"`
	Ton_balance      float64            `bson:"ton_balance" json:"ton_balance"`
	M5_balance       float64            `bson:"m5_balance" json:"m5_balance"`
	Dfc_balance      float64            `bson:"dfc_balance" json:"dfc_balance"`
	Cubes            int                `bson:"cubes" json:"cubes"`
	ReferralCode     string             `bson:"referral_code" json:"referral_code"`
	ReferredBy       string             `bson:"referred_by" json:"referred_by,omitempty"`
	ReferralEarnings map[string]float64 `bson:"referral_earnings" json:"referral_earnings,omitempty"`
	Points           float64            `bson:"points" json:"points"`
	TgID             string             `bson:"tgid" json:"tgid"`
	Language         string             `bson:"language" json:"language"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}
