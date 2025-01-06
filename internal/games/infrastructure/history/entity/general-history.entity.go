package entities

import "time"

type GameRecord struct {
	Player1Name     string    `bson:"player1_name" json:"Player1Name"`
	Player2Name     string    `bson:"player2_name" json:"Player2Name"`
	Player1Score    int       `bson:"player1_score" json:"Player1Score"`
	Player2Score    int       `bson:"player2_score" json:"Player2Score"`
	Winner          string    `bson:"winner" json:"Winner"`
	Player1Earnings float64   `bson:"player1_earnings" json:"Player1Earnings"`
	Player2Earnings float64   `bson:"player2_earnings" json:"Player2Earnings"`
	TimePlayed      time.Time `bson:"time_played" json:"TimePlayed"`
	TokenType       string    `bson:"token_type" json:"TokenType"`
	BetAmount       float64   `bson:"bet_amount" json:"BetAmount"`
	Player1Wallet   string    `bson:"player1_wallet" json:"Player1Wallet"`
	Player2Wallet   string    `bson:"player2_wallet" json:"Player2Wallet"`
	Counter         int       `bson:"counter" json:"Counter"` // Инкрементируемое поле
}
