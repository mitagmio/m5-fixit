package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PromoCodeStatus represents the current status of a promocode
type PromoCodeStatus string

const (
	Active   PromoCodeStatus = "active"   // Promocode is active and can be used
	Expired  PromoCodeStatus = "expired"  // Promocode has expired
	Depleted PromoCodeStatus = "depleted" // Promocode has reached max activations
)

// PromoCodeEntity represents the structure of a promocode in the database
type PromoCodeEntity struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	Code             string             `bson:"code"`                 // Unique code for the promocode
	TokenType        string             `bson:"token_type"`           // Reward type
	Amount           float64            `bson:"amount"`               // Reward amount
	MaxActivations   int                `bson:"max_activations"`      // Maximum activations allowed
	UsedActivations  int                `bson:"used_activations"`     // Number of times the promocode has been used
	ActivatedWallets []string           `bson:"activated_wallets"`    // List of wallets that have activated the promocode
	Status           PromoCodeStatus    `bson:"status"`               // Current status
	ExpiresAt        *time.Time         `bson:"expires_at,omitempty"` // Expiration date
	CreatedAt        time.Time          `bson:"created_at"`           // Creation timestamp
	UpdatedAt        time.Time          `bson:"updated_at"`           // Last update timestamp
}
