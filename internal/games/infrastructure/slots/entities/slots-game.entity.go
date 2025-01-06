package entities

import "time"

// SlotGame - Сущность, хранящая информацию о сыгранной игре.
type SlotGame struct {
	Wallet    string    `bson:"wallet"`     // Кошелек игрока
	Bet       float64   `bson:"bet"`        // Ставка игрока
	Result    string    `bson:"result"`     // Результат игры (например, "win", "lose")
	WinAmount float64   `bson:"win_amount"` // Сумма выигрыша (если был выигрыш)
	PlayedAt  time.Time `bson:"played_at"`  // Время игры
}
