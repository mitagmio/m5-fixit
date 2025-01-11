package entities

import "time"

type GameHistoryItem struct {
	ID          string    `json:"id" bson:"_id,omitempty"`
	Player1Name string    `json:"player1_name" bson:"player1_name"`
	Player2Name string    `json:"player2_name" bson:"player2_name"`
	Winner      string    `json:"winner" bson:"winner"`
	BetAmount   float64   `json:"bet_amount" bson:"bet_amount"`
	TokenType   string    `json:"token_type" bson:"token_type"`
	GameType    string    `json:"game_type" bson:"game_type"`
	TimePlayed  time.Time `json:"time_played" bson:"time_played"`
}
