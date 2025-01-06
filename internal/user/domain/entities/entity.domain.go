package entities

type Withdrawal struct {
	ID         string  `json:"id"`
	Amount     float64 `json:"amount"`
	Wallet     string  `json:"wallet"`
	JettonName string  `json:"jetton_name"`
	CreatedAt  string  `json:"created_at"`
}
