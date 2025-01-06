package entities

import "time"

type User struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	FirstName        string             `json:"first_name"` // Новое поле FirstName
	Wallet           string             `json:"wallet"`
	Ton_balance      float64            `json:"ton_balance"`
	M5_balance       float64            `json:"m5_balance"`
	Dfc_balance      float64            `json:"dfc_balance"`
	Cubes            int                `json:"cubes"`
	ReferralCode     string             `json:"referral_code"`
	ReferredBy       string             `json:"referred_by"`
	ReferralEarnings map[string]float64 `json:"referral_earnings"`
	Points           float64            `json:"points"`
	TgID             string             `json:"tgid"`
	Language         string             `json:"language"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}
