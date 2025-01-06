package entities

import "time"

type BotBalanceEntity struct {
	TonBalance float64   `json:"ton_balance" bson:"ton_balance"`
	M5Balance  float64   `json:"m5_balance" bson:"m5_balance"`
	DfcBalance float64   `json:"dfc_balance" bson:"dfc_balance"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" bson:"updated_at"`
}
