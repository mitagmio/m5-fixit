package entities

import "time"

// SlotsBalance - Сущность, хранящая баланс пользователя в тоннах и кубах.
type SlotsBalance struct {
	Tons      float64   `bson:"tons"`       // Баланс в тоннах
	Cubes     float64   `bson:"cubes"`      // Баланс в кубах
	UpdatedAt time.Time `bson:"updated_at"` // Время последнего обновления
}
