package services

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"

	userRepo "github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
	"github.com/gorilla/websocket"
)

// DiceGameService отвечает за логику игры Dice PvP
type DiceGameService struct {
	mu       sync.Mutex
	lobbies  map[string]*Lobby
	userRepo *userRepo.UserRepository
}

// Lobby представляет игровую комнату
type Lobby struct {
	ID           string
	Player1      *Player
	Player2      *Player
	TargetScore  int
	Status       string // "waiting", "in_progress", "finished"
	CurrentTurn  string // "player1" или "player2"
	CurrentRound int
	RoundRolls   map[string]int
	TokenType    string  // Тип токена ("dfc", "m5", "ton")
	BetAmount    float64 // Сумма ставки
}

// Player представляет игрока
type Player struct {
	ID            string
	Wallet        string // Кошелек пользователя
	Conn          *websocket.Conn
	Score         int
	TokenBalances map[string]float64 // Баланс токенов по типам
}

// RoundResult представляет результаты раунда
type RoundResult struct {
	Round        int    `json:"round"`
	Player1Roll  int    `json:"player1_roll"`
	Player2Roll  int    `json:"player2_roll"`
	Player1Score int    `json:"player1_score"`
	Player2Score int    `json:"player2_score"`
	NextTurn     string `json:"next_turn"`
	GameOver     bool   `json:"game_over"`
	Winner       string `json:"winner,omitempty"`
}

func NewDiceGameService(userRepo *userRepo.UserRepository) *DiceGameService {
	rand.Seed(int64(rand.Intn(1000000))) // Инициализация генератора случайных чисел
	log.Println("DiceGameService initialized.")
	return &DiceGameService{
		lobbies:  make(map[string]*Lobby),
		userRepo: userRepo,
	}
}

// CreateLobby создает новое лобби
func (s *DiceGameService) CreateLobby(player *Player, targetScore int, tokenType string, betAmount float64) (string, error) {
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		log.Printf("[CreateLobby] Invalid token type: %s", tokenType)
		return "", fmt.Errorf("invalid token type: %s", tokenType)
	}

	lobbyID := fmt.Sprintf("%06d", rand.Intn(1000000))
	s.mu.Lock()
	s.lobbies[lobbyID] = &Lobby{
		ID:           lobbyID,
		Player1:      player,
		TargetScore:  targetScore,
		Status:       "waiting",
		CurrentRound: 1,
		RoundRolls:   make(map[string]int),
		TokenType:    tokenType,
		BetAmount:    betAmount,
	}
	s.mu.Unlock()

	log.Printf("[CreateLobby] Lobby created. ID: %s, Player1: %s, TokenType: %s, BetAmount: %.2f, TargetScore: %d", lobbyID, player.ID, tokenType, betAmount, targetScore)
	return lobbyID, nil
}

func (s *DiceGameService) JoinLobby(player *Player, lobbyID, wallet, tokenType string) error {

	if player == nil {
		log.Println("[JoinLobby] Player object is nil.")
		return fmt.Errorf("player object is nil")
	}

	if wallet == "" {
		log.Println("[JoinLobby] Wallet address is required.")
		return fmt.Errorf("wallet address is required")
	}

	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		log.Printf("[JoinLobby] Invalid token type: %s", tokenType)
		return fmt.Errorf("invalid token type: %s", tokenType)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	lobby, exists := s.lobbies[lobbyID]
	if !exists {
		log.Printf("[JoinLobby] Lobby not found: %s", lobbyID)
		return fmt.Errorf("lobby not found or already in progress")
	}

	if lobby.Status != "waiting" {
		log.Printf("[JoinLobby] Lobby is not waiting. Current status: %s", lobby.Status)
		return fmt.Errorf("lobby not found or already in progress")
	}

	if tokenType != lobby.TokenType {
		log.Printf("[JoinLobby] Token type mismatch. Expected: %s, Got: %s", lobby.TokenType, tokenType)
		return fmt.Errorf("mismatched token type: expected %s, got %s", lobby.TokenType, tokenType)
	}

	hasBalance, err := s.userRepo.HasSufficientBalance(context.Background(), wallet, tokenType, lobby.BetAmount)
	if err != nil {
		log.Printf("[JoinLobby] Error checking balance for wallet %s: %v", wallet, err)
		return fmt.Errorf("error checking balance: %v", err)
	}
	if !hasBalance {
		log.Printf("[JoinLobby] Insufficient balance for wallet: %s", wallet)
		return fmt.Errorf("insufficient balance for the bet")
	}

	lobby.Player2 = player
	lobby.Status = "in_progress"
	lobby.CurrentTurn = "player1"
	lobby.RoundRolls = make(map[string]int)

	log.Printf("[JoinLobby] Player %s joined lobby %s. Game in progress.", wallet, lobbyID)
	return nil
}

func (s *DiceGameService) RollDice(player *Player, lobbyID string) (RoundResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверка существования лобби
	lobby, exists := s.lobbies[lobbyID]
	if !exists || lobby.Status != "in_progress" {
		log.Printf("[RollDice] Lobby not found or not in progress. LobbyID: %s, Status: %s", lobbyID, lobby.Status)
		return RoundResult{}, fmt.Errorf("лобби не найдено или игра не начата")
	}

	log.Printf("[RollDice] Player %s attempting to roll in Lobby %s. Current turn: %s", player.ID, lobbyID, lobby.CurrentTurn)

	// Проверка очереди
	if (lobby.CurrentTurn == "player1" && player.ID != lobby.Player1.ID) ||
		(lobby.CurrentTurn == "player2" && player.ID != lobby.Player2.ID) {
		log.Printf("[RollDice] Player %s attempted to roll out of turn. Current turn: %s", player.ID, lobby.CurrentTurn)
		return RoundResult{}, fmt.Errorf("не ваша очередь")
	}

	// Бросок кубика
	roll := rand.Intn(6) + 1
	player.Score += roll
	lobby.RoundRolls[player.ID] = roll
	log.Printf("[RollDice] Player %s rolled %d. Total score: %d", player.ID, roll, player.Score)

	// Проверяем завершение раунда
	if len(lobby.RoundRolls) == 2 {
		result := RoundResult{
			Round:        lobby.CurrentRound,
			Player1Roll:  lobby.RoundRolls[lobby.Player1.ID],
			Player2Roll:  lobby.RoundRolls[lobby.Player2.ID],
			Player1Score: lobby.Player1.Score,
			Player2Score: lobby.Player2.Score,
		}

		log.Printf("[RollDice] Round %d completed. Player1 Score: %d, Player2 Score: %d", lobby.CurrentRound, lobby.Player1.Score, lobby.Player2.Score)

		// Проверка завершения игры
		if lobby.Player1.Score >= lobby.TargetScore || lobby.Player2.Score >= lobby.TargetScore {
			if lobby.Player1.Score > lobby.Player2.Score {
				result.Winner = "player1"
			} else if lobby.Player2.Score > lobby.Player1.Score {
				result.Winner = "player2"
			} else {
				result.Winner = "draw"
			}
			result.GameOver = true
			lobby.Status = "finished"

			log.Printf("[RollDice] Game over. Winner: %s", result.Winner)
			return result, nil
		}

		// Сброс текущего раунда
		lobby.RoundRolls = make(map[string]int)
		lobby.CurrentTurn = s.getNextTurn(lobby.CurrentTurn)
		lobby.CurrentRound++
		log.Printf("[RollDice] Round %d starts. Next turn: %s", lobby.CurrentRound, lobby.CurrentTurn)
		return result, nil
	}

	// Ход не завершен
	lobby.CurrentTurn = s.getNextTurn(lobby.CurrentTurn)
	log.Printf("[RollDice] Turn switched. Next turn: %s", lobby.CurrentTurn)
	return RoundResult{
		Round:        lobby.CurrentRound,
		Player1Roll:  lobby.RoundRolls[lobby.Player1.ID],
		Player2Roll:  lobby.RoundRolls[lobby.Player2.ID],
		Player1Score: lobby.Player1.Score,
		Player2Score: lobby.Player2.Score,
		NextTurn:     lobby.CurrentTurn,
		GameOver:     false,
	}, nil
}

func (s *DiceGameService) getNextTurn(currentTurn string) string {
	if currentTurn == "player1" {
		return "player2"
	}
	return "player1"
}

func (s *DiceGameService) GetLobby(lobbyID string) *Lobby {
	s.mu.Lock()
	defer s.mu.Unlock()

	lobby, exists := s.lobbies[lobbyID]
	if !exists {
		log.Printf("[GetLobby] Lobby not found: %s", lobbyID)
		return nil
	}

	log.Printf("[GetLobby] Lobby found: %+v", lobby)
	return lobby
}

func (s *DiceGameService) GetAvailableLobbies() []map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	var availableLobbies []map[string]interface{}
	for id, lobby := range s.lobbies {
		if lobby.Status == "waiting" && lobby.Player2 == nil {
			availableLobbies = append(availableLobbies, map[string]interface{}{
				"lobby_id":   id,
				"player1":    lobby.Player1.ID,
				"token_type": lobby.TokenType,
				"bet_amount": lobby.BetAmount,
			})
		}
	}

	log.Printf("[GetAvailableLobbies] Found %d available lobbies.", len(availableLobbies))
	return availableLobbies
}

func (s *DiceGameService) CloseLobby(player *Player, lobbyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lobby, exists := s.lobbies[lobbyID]
	if !exists {
		return fmt.Errorf("lobby not found")
	}

	if lobby.Player1.ID != player.ID {
		return fmt.Errorf("only the creator can close the lobby")
	}

	// Удаляем лобби
	delete(s.lobbies, lobbyID)
	log.Printf("[CloseLobby] Lobby %s closed by player %s", lobbyID, player.ID)
	return nil
}
