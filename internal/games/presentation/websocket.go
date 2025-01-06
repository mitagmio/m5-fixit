package presentation

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	referralServices "github.com/Peranum/tg-dice/internal/referral/domain/services"
	"github.com/Peranum/tg-dice/internal/user/infrastructure/repositories"
	"github.com/gorilla/websocket"

	// Наш сервис для сохранения истории игр
	gameServices "github.com/Peranum/tg-dice/internal/games/domain/history/services"
)

// =======================================
// PvP Game Structures and Service
// =======================================

type Lobby struct {
	ID           string
	Player1      *Player
	Player2      *Player
	TargetScore  int
	Status       string
	CurrentRound int
	RoundRolls   map[string]int
	TokenType    string
	BetAmount    float64
	CurrentTurn  string
	ReadyPlayer1 bool
	ReadyPlayer2 bool
}

type Player struct {
	ID            string
	Wallet        string
	FirstName     string
	Conn          *websocket.Conn
	Score         int
	TokenBalances map[string]float64
}

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

type DicePVPGameService struct {
	lobbies   map[string]*Lobby
	lobbiesMu sync.Mutex
	clients   map[*websocket.Conn]bool
	clientsMu sync.Mutex
	upgrader  websocket.Upgrader
	userRepo  *repositories.UserRepository

	// Внедряем GameService, чтобы сохранять записи об играх
	gameService *gameServices.GameService
}

// =======================================
// Конструктор
// =======================================
func NewDicePVPGameService(
	userRepo *repositories.UserRepository,
	gameService *gameServices.GameService,
) *DicePVPGameService {
	return &DicePVPGameService{
		lobbies: make(map[string]*Lobby),
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		userRepo:    userRepo,
		gameService: gameService,
	}
}

// recoverPanic — вспомогательная функция для отлова паник
func recoverPanic() {
	if r := recover(); r != nil {
		log.Printf("[PANIC RECOVERED] %v", r)
	}
}

// настройка пинг/понг
func (s *DicePVPGameService) setPingPongHandlers(conn *websocket.Conn) {
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		log.Println("[setPingPongHandlers] Получен pong от клиента")
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
}

// startPingRoutine — периодически отправляет пинг-сообщения, чтобы поддерживать активное соединение
func (s *DicePVPGameService) startPingRoutine(conn *websocket.Conn, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			log.Println("[startPingRoutine] Отправка ping клиенту")
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[startPingRoutine] Ошибка отправки ping: %v", err)
				return
			}
		}
	}()
}

// writeJSON с тайм-аутом
func (s *DicePVPGameService) safeWriteJSON(conn *websocket.Conn, v interface{}) error {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := conn.WriteJSON(v)
	if err != nil {
		log.Printf("[safeWriteJSON] Ошибка при отправке JSON: %v", err)
	}
	return err
}

func (s *DicePVPGameService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Println("[HandleWebSocket] Инициализация нового WebSocket-соединения")
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[HandleWebSocket] Ошибка при обновлении WebSocket: %v", err)
		return
	}
	defer func() {
		conn.Close()
		log.Println("[HandleWebSocket] WebSocket-соединение закрыто")
	}()

	s.setPingPongHandlers(conn)
	s.startPingRoutine(conn, 30*time.Second)

	s.addClient(conn)
	defer s.removeClient(conn)

	defer recoverPanic() // Ловим паники в этой горутине

	var player *Player
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("[HandleWebSocket] Ошибка при чтении JSON: %v", err)
			return
		}

		action, ok := message["action"].(string)
		if !ok {
			log.Printf("[HandleWebSocket] Неверный формат действия: %#v", message)
			s.safeWriteJSON(conn, map[string]interface{}{
				"action":  "error",
				"message": "Неверный формат действия",
			})
			continue
		}

		log.Printf("[HandleWebSocket] Получено действие: %s, данные: %#v", action, message)

		switch action {
		case "create_lobby":
			s.handleCreateLobby(conn, message, &player)
		case "join_lobby":
			s.handleJoinLobby(conn, message, &player)
		case "roll_dice":
			s.handleRollDice(conn, message, player)
		case "list_lobbies":
			s.sendLobbyList(conn)
		case "terminate_game":
			s.handleTerminateGame(conn, message, player)
		case "delete_lobby":
			s.handleDeleteLobby(conn, message, player)
		case "confirm_ready":
			s.handleConfirmReady(conn, message, player)
		default:
			log.Printf("[HandleWebSocket] Неизвестное действие: %s", action)
			s.safeWriteJSON(conn, map[string]interface{}{
				"action":  "error",
				"message": "Неизвестное действие",
			})
		}
	}
}

func (s *DicePVPGameService) addClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.clients[conn] = true
	log.Printf("[addClient] Новый клиент подключён: %v", conn.RemoteAddr())
}

func (s *DicePVPGameService) removeClient(conn *websocket.Conn) {
	var removedLobbyID string

	s.lobbiesMu.Lock()
	for lobbyID, lobby := range s.lobbies {
		if lobby.Player1 != nil && lobby.Player1.Conn == conn && lobby.Status == "waiting" {
			delete(s.lobbies, lobbyID)
			log.Printf("[removeClient] Лобби %s удалено, так как создатель отключился", lobbyID)
			removedLobbyID = lobbyID
			break // Прекратить дальнейший поиск
		}
	}
	s.lobbiesMu.Unlock()

	if removedLobbyID != "" {
		s.BroadcastLobbyList() // Обновление списка лобби без блокировки
	}
}

// =======================================
// Обработка создания лобби
// =======================================
func (s *DicePVPGameService) handleCreateLobby(conn *websocket.Conn, message map[string]interface{}, player **Player) {
	log.Println("[handleCreateLobby] Начало обработки создания лобби")

	targetScore, ok := getInt(message, "target_score", 25)
	if !ok {
		log.Println("[handleCreateLobby] Ошибка: отсутствует или неверный target_score")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Неверный или отсутствующий target_score",
		})
		return
	}

	tokenType, ok := message["token_type"].(string)
	if !ok || tokenType == "" {
		log.Println("[handleCreateLobby] Ошибка: отсутствует или неверный token_type")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Неверный или отсутствующий token_type",
		})
		return
	}

	betAmount, ok := getFloat64(message, "bet_amount", 0)
	if !ok || betAmount <= 0 {
		log.Println("[handleCreateLobby] Ошибка: отсутствует или неверный bet_amount")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Неверный или отсутствующий bet_amount",
		})
		return
	}

	wallet, ok := message["wallet"].(string)
	if !ok || wallet == "" {
		log.Println("[handleCreateLobby] Ошибка: отсутствует wallet")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Кошелёк пользователя отсутствует",
		})
		return
	}

	firstName, _ := message["first_name"].(string)
	if firstName == "" {
		firstName = "Player"
	}

	*player = &Player{
		ID:            generatePlayerID(),
		Wallet:        wallet,
		FirstName:     firstName,
		Conn:          conn,
		TokenBalances: make(map[string]float64),
	}

	log.Printf("[handleCreateLobby] Перед созданием лобби. PlayerID: %s, Name: %s, Wallet: %s, TargetScore: %d, TokenType: %s, BetAmount: %.2f",
		(*player).ID, (*player).FirstName, (*player).Wallet, targetScore, tokenType, betAmount)

	lobbyID, err := s.CreateLobby(*player, targetScore, tokenType, betAmount)
	if err != nil {
		log.Printf("[handleCreateLobby] Ошибка создания лобби: %v", err)
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": err.Error(),
		})
		return
	}

	log.Printf("[handleCreateLobby] Лобби создано успешно: %s", lobbyID)
	s.safeWriteJSON(conn, map[string]interface{}{
		"action":       "lobby_created",
		"lobby_id":     lobbyID,
		"token_type":   tokenType,
		"bet_amount":   betAmount,
		"target_score": targetScore,
	})

	s.BroadcastLobbyList()
}

// =======================================
// Обработка присоединения к лобби
// =======================================
func (s *DicePVPGameService) handleJoinLobby(conn *websocket.Conn, message map[string]interface{}, player **Player) {
	log.Println("[handleJoinLobby] Начало обработки присоединения к лобби")

	lobbyID, ok := message["lobby_id"].(string)
	if !ok || lobbyID == "" {
		log.Println("[handleJoinLobby] Ошибка: отсутствует или неверный lobby_id")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Неверный ID лобби",
		})
		return
	}

	wallet, ok := message["wallet"].(string)
	if !ok || wallet == "" {
		log.Println("[handleJoinLobby] Ошибка: отсутствует wallet")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Кошелёк пользователя отсутствует",
		})
		return
	}

	firstName, _ := message["first_name"].(string)
	if firstName == "" {
		firstName = "Player"
	}

	*player = &Player{
		ID:            generatePlayerID(),
		Wallet:        wallet,
		FirstName:     firstName,
		Conn:          conn,
		TokenBalances: make(map[string]float64),
	}

	log.Printf("[handleJoinLobby] Перед присоединением к лобби. PlayerID: %s, Name: %s, Wallet: %s, LobbyID: %s",
		(*player).ID, (*player).FirstName, (*player).Wallet, lobbyID)

	err := s.JoinLobby(*player, lobbyID)
	if err != nil {
		log.Printf("[handleJoinLobby] Ошибка присоединения к лобби: %v", err)
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": err.Error(),
		})
		return
	}

	log.Printf("[handleJoinLobby] Игрок %s (%s) успешно присоединился к лобби %s", (*player).ID, (*player).FirstName, lobbyID)
	s.safeWriteJSON(conn, map[string]interface{}{
		"action":   "joined_lobby",
		"lobby_id": lobbyID,
	})
	s.BroadcastLobbyList()
}

// =======================================
// Обработка броска кубиков (RollDice)
// =======================================
func (s *DicePVPGameService) handleRollDice(conn *websocket.Conn, message map[string]interface{}, player *Player) {
	log.Println("[handleRollDice] Начало обработки броска кубиков")

	if player == nil {
		log.Println("[handleRollDice] Ошибка: игрок не присоединился к лобби")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Вы не присоединились к лобби",
		})
		return
	}

	lobbyID, ok := message["lobby_id"].(string)
	if !ok || lobbyID == "" {
		log.Println("[handleRollDice] Ошибка: отсутствует или неверный lobby_id")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Неверный ID лобби",
		})
		return
	}

	log.Printf("[handleRollDice] Игрок %s (%s) бросает кости в лобби %s",
		player.ID, player.FirstName, lobbyID)
	s.RollDice(player, lobbyID)
}

// =======================================
// Обработка досрочного завершения игры
// =======================================
func (s *DicePVPGameService) handleTerminateGame(conn *websocket.Conn, message map[string]interface{}, player *Player) {
	log.Println("[handleTerminateGame] Начало обработки запроса на досрочное завершение игры")

	if player == nil {
		log.Println("[handleTerminateGame] Игрок не присоединился к лобби")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Вы не участвуете в лобби",
		})
		return
	}

	lobbyID, ok := message["lobby_id"].(string)
	if !ok || lobbyID == "" {
		log.Println("[handleTerminateGame] Ошибка: отсутствует lobby_id")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Неверный ID лобби",
		})
		return
	}

	winner, ok := message["winner"].(string)
	if !ok || (winner != "player1" && winner != "player2") {
		log.Println("[handleTerminateGame] Ошибка: отсутствует или неверный идентификатор победителя")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Неверный идентификатор победителя",
		})
		return
	}

	log.Printf("[handleTerminateGame] Запрос на завершение игры в лобби %s с победителем %s от игрока %s (%s)",
		lobbyID, winner, player.ID, player.FirstName)

	err := s.TerminateGame(player, lobbyID, winner)
	if err != nil {
		log.Printf("[handleTerminateGame] Ошибка завершения игры: %v", err)
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": err.Error(),
		})
		return
	}

	log.Printf("[handleTerminateGame] Игра %s завершена досрочно. Победитель: %s", lobbyID, winner)
	s.safeWriteJSON(conn, map[string]interface{}{
		"action":   "game_terminated",
		"lobby_id": lobbyID,
		"winner":   winner, // Мы изменим это ниже
	})

	s.BroadcastLobbyList()
}

// =======================================
// Обработка удаления лобби
// =======================================
func (s *DicePVPGameService) handleDeleteLobby(conn *websocket.Conn, message map[string]interface{}, player *Player) {
	log.Println("[handleDeleteLobby] Начало обработки удаления лобби")

	if player == nil {
		log.Println("[handleDeleteLobby] Игрок не инициализирован")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Вы не присоединились к лобби",
		})
		return
	}

	lobbyID, ok := message["lobby_id"].(string)
	if !ok || lobbyID == "" {
		log.Println("[handleDeleteLobby] Ошибка: отсутствует или неверный lobby_id")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Неверный ID лобби",
		})
		return
	}

	log.Printf("[handleDeleteLobby] Запрос на удаление лобби: %s от игрока %s (%s)",
		lobbyID, player.ID, player.FirstName)

	err := s.DeleteLobby(player, lobbyID)
	if err != nil {
		log.Printf("[handleDeleteLobby] Ошибка удаления лобби: %v", err)
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": err.Error(),
		})
		return
	}

	s.safeWriteJSON(conn, map[string]interface{}{
		"action":   "lobby_deleted",
		"lobby_id": lobbyID,
	})

	log.Printf("[handleDeleteLobby] Лобби %s успешно удалено создателем", lobbyID)
	s.BroadcastLobbyList()
}

func (s *DicePVPGameService) DeleteLobby(player *Player, lobbyID string) error {
	log.Printf("[DeleteLobby] Попытка удаления лобби: %s", lobbyID)
	s.lobbiesMu.Lock()
	defer s.lobbiesMu.Unlock()

	lobby, exists := s.lobbies[lobbyID]
	if !exists {
		log.Printf("[DeleteLobby] Лобби %s не найдено", lobbyID)
		return fmt.Errorf("лобби не найдено")
	}

	if lobby.Player1 != player {
		log.Printf("[DeleteLobby] Игрок %s (%s) не является создателем лобби %s",
			player.ID, player.FirstName, lobbyID)
		return fmt.Errorf("у вас нет прав на удаление этого лобби")
	}

	if lobby.Status != "waiting" {
		log.Printf("[DeleteLobby] Лобби %s уже в статусе %s, удаление невозможно", lobbyID, lobby.Status)
		return fmt.Errorf("игра уже началась, удаление лобби невозможно")
	}

	delete(s.lobbies, lobbyID)
	log.Printf("[DeleteLobby] Лобби %s удалено", lobbyID)
	return nil
}

// =======================================
// Обработка подтверждения готовности
// =======================================
func (s *DicePVPGameService) handleConfirmReady(conn *websocket.Conn, message map[string]interface{}, player *Player) {
	log.Println("[handleConfirmReady] Обработка подтверждения готовности")

	if player == nil {
		log.Println("[handleConfirmReady] Ошибка: игрок не присоединился к лобби")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Вы не присоединились к лобби",
		})
		return
	}

	lobbyID, ok := message["lobby_id"].(string)
	if !ok || lobbyID == "" {
		log.Println("[handleConfirmReady] Ошибка: отсутствует или неверный lobby_id")
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": "Неверный ID лобби",
		})
		return
	}

	err := s.ConfirmReady(player, lobbyID)
	if err != nil {
		s.safeWriteJSON(conn, map[string]interface{}{
			"action":  "error",
			"message": err.Error(),
		})
		return
	}

	s.safeWriteJSON(conn, map[string]interface{}{
		"action":  "ready_confirmation",
		"message": "Вы подтвердили свою готовность",
	})
}

// =======================================
// Реализации игровых методов
// =======================================
func (s *DicePVPGameService) CreateLobby(player *Player, targetScore int, tokenType string, betAmount float64) (string, error) {
	log.Printf("[CreateLobby] Проверка валидности токена: %s", tokenType)
	validTokens := map[string]bool{
		"ton_balance": true,
		"m5_balance":  true,
		"dfc_balance": true,
	}

	if !validTokens[tokenType] {
		log.Printf("[CreateLobby] Неверный тип токена: %s", tokenType)
		return "", fmt.Errorf("недопустимый тип токена: %s", tokenType)
	}

	log.Printf("[CreateLobby] Перед проверкой баланса в БД для кошелька: %s", player.Wallet)
	ctx, cancel := s.withDBTimeout()
	defer cancel()

	sufficient, err := s.userRepo.HasSufficientBalance(ctx, player.Wallet, tokenType, betAmount)
	if err != nil {
		log.Printf("[CreateLobby] Ошибка проверки баланса: %v", err)
		return "", fmt.Errorf("ошибка проверки баланса: %v", err)
	}

	if !sufficient {
		log.Printf("[CreateLobby] Недостаточно средств для создания лобби: wallet=%s", player.Wallet)
		return "", fmt.Errorf("недостаточно средств для создания лобби")
	}

	lobbyID := generateLobbyID()
	log.Printf("[CreateLobby] Сгенерирован ID лобби: %s", lobbyID)

	s.lobbiesMu.Lock()
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
	s.lobbiesMu.Unlock()

	log.Printf("[CreateLobby] Лобби создано: %s", lobbyID)
	return lobbyID, nil
}

func (s *DicePVPGameService) JoinLobby(player *Player, lobbyID string) error {
	log.Printf("[JoinLobby] Поиск лобби %s", lobbyID)
	s.lobbiesMu.Lock()
	lobby, exists := s.lobbies[lobbyID]
	s.lobbiesMu.Unlock()

	if !exists || lobby.Status != "waiting" {
		log.Printf("[JoinLobby] Лобби не найдено или уже началась игра: %s", lobbyID)
		return fmt.Errorf("лобби не найдено или уже началась игра")
	}

	ctx, cancel := s.withDBTimeout()
	defer cancel()

	sufficient, err := s.userRepo.HasSufficientBalance(ctx, player.Wallet, lobby.TokenType, lobby.BetAmount)
	if err != nil {
		log.Printf("[JoinLobby] Ошибка проверки баланса: %v", err)
		return fmt.Errorf("ошибка проверки баланса: %v", err)
	}

	if !sufficient {
		log.Printf("[JoinLobby] Недостаточно средств у кошелька: %s", player.Wallet)
		return fmt.Errorf("недостаточно средств для присоединения к лобби")
	}

	s.lobbiesMu.Lock()
	lobby, exists = s.lobbies[lobbyID]
	if !exists || lobby.Status != "waiting" {
		s.lobbiesMu.Unlock()
		log.Printf("[JoinLobby] Лобби не найдено или уже началась игра при повторном доступе: %s", lobbyID)
		return fmt.Errorf("лобби не найдено или уже началась игра")
	}

	lobby.Player2 = player
	lobby.Status = "in_progress"
	lobby.CurrentTurn = "player1"
	lobby.RoundRolls = make(map[string]int)

	player1Conn := lobby.Player1.Conn
	player2Conn := lobby.Player2.Conn

	// Сообщение для Player1
	startMessagePlayer1 := map[string]interface{}{
		"action":        "game_start",
		"message":       "Игра начинается! Первый ход за Игроком 1.",
		"current_turn":  lobby.CurrentTurn,
		"player_id":     "player1",
		"player_name":   lobby.Player1.FirstName,
		"lobby_id":      lobby.ID,
		"target_score":  lobby.TargetScore,
		"current_round": lobby.CurrentRound,
		"token_type":    lobby.TokenType,
		"bet_amount":    lobby.BetAmount,
		"player1_id":    lobby.Player1.ID,        // Добавлено: ID Player1
		"player2_id":    lobby.Player2.ID,        // Добавлено: ID Player2
		"player1_name":  lobby.Player1.FirstName, // Добавлено: Имя Player1
		"player2_name":  lobby.Player2.FirstName, // Добавлено: Имя Player2
	}
	// Сообщение для Player2
	startMessagePlayer2 := map[string]interface{}{
		"action":        "game_start",
		"message":       "Игра начинается! Первый ход за Игроком 1.",
		"current_turn":  lobby.CurrentTurn,
		"player_id":     "player2",
		"player_name":   lobby.Player2.FirstName,
		"lobby_id":      lobby.ID,
		"target_score":  lobby.TargetScore,
		"current_round": lobby.CurrentRound,
		"token_type":    lobby.TokenType,
		"bet_amount":    lobby.BetAmount,
		"player1_id":    lobby.Player1.ID,        // Добавлено: ID Player1
		"player2_id":    lobby.Player2.ID,        // Добавлено: ID Player2
		"player1_name":  lobby.Player1.FirstName, // Добавлено: Имя Player1
		"player2_name":  lobby.Player2.FirstName, // Добавлено: Имя Player2
	}

	s.lobbiesMu.Unlock()

	err1 := s.safeWriteJSON(player1Conn, startMessagePlayer1)
	err2 := s.safeWriteJSON(player2Conn, startMessagePlayer2)
	if err1 != nil || err2 != nil {
		log.Printf("[JoinLobby] Ошибка отправки game_start: %v, %v", err1, err2)
		return fmt.Errorf("не удалось уведомить игроков о начале игры")
	}

	log.Printf("[JoinLobby] Уведомления о старте игры отправлены в лобби %s", lobbyID)
	return nil
}

// =======================================
// RollDice: исправленный вызов SaveGame
// =======================================
func (s *DicePVPGameService) RollDice(player *Player, lobbyID string) {
	log.Printf("[RollDice] Попытка броска: PlayerID=%s, FirstName=%s, LobbyID=%s",
		player.ID, player.FirstName, lobbyID)

	s.lobbiesMu.Lock()
	lobby, exists := s.lobbies[lobbyID]
	if !exists || lobby.Status != "in_progress" {
		s.lobbiesMu.Unlock()
		log.Println("[RollDice] Лобби не найдено или не в процессе")
		s.safeWriteJSON(player.Conn, map[string]interface{}{
			"action":  "error",
			"message": "Лобби не найдено или игра не в процессе",
		})
		return
	}

	var playerKey string
	if lobby.Player1 == player {
		playerKey = "player1"
	} else if lobby.Player2 == player {
		playerKey = "player2"
	} else {
		s.lobbiesMu.Unlock()
		log.Println("[RollDice] Игрок не участвует в лобби")
		s.safeWriteJSON(player.Conn, map[string]interface{}{
			"action":  "error",
			"message": "Вы не участвуете в этом лобби",
		})
		return
	}

	if lobby.CurrentTurn != playerKey {
		s.lobbiesMu.Unlock()
		log.Printf("[RollDice] Не ваш ход (%s), текущий: %s", playerKey, lobby.CurrentTurn)
		s.safeWriteJSON(player.Conn, map[string]interface{}{
			"action":  "error",
			"message": "Сейчас не ваш ход",
		})
		return
	}

	// Бросок кубиков
	roll1 := rand.Intn(6) + 1
	roll2 := rand.Intn(6) + 1
	totalRoll := roll1 + roll2
	lobby.RoundRolls[playerKey] = totalRoll

	// Проверка на дубль и начисление бонуса
	bonus := 0
	if roll1 == roll2 {
		bonus = 1 // Бонус за дубль
		log.Printf("[RollDice] Игрок %s получил бонус за дубль! Roll: %d-%d", player.ID, roll1, roll2)
	}

	// Обновление счета игрока с бонусом
	player.Score += totalRoll + bonus
	log.Printf("[RollDice] Игрок %s (%s) бросил: %d и %d (сумма: %d, бонус: %d) в лобби %s",
		player.ID, player.FirstName, roll1, roll2, totalRoll, bonus, lobbyID)

	partialResultMessage := map[string]interface{}{
		"action":        "partial_round_result",
		"round":         lobby.CurrentRound,
		"player":        player.ID,
		"player_name":   player.FirstName, // Используем FirstName
		"roll1":         roll1,
		"roll2":         roll2,
		"total_roll":    totalRoll,
		"bonus":         bonus, // Бонус за дубль
		"player1_score": lobby.Player1.Score,
		"player2_score": lobby.Player2.Score,
		"player1_name":  lobby.Player1.FirstName,
		"player2_name":  lobby.Player2.FirstName,
	}

	player1Conn := lobby.Player1.Conn
	player2Conn := lobby.Player2.Conn
	log.Println("[RollDice] Отправка partial_round_result игрокам")
	s.safeWriteJSON(player1Conn, partialResultMessage)
	s.safeWriteJSON(player2Conn, partialResultMessage)

	// Проверяем, завершили ли оба игрока свой ход в текущем раунде
	if len(lobby.RoundRolls) == 2 {
		log.Printf("[RollDice] Раунд %d завершен", lobby.CurrentRound)

		// Проверяем, достиг ли кто-то из игроков TargetScore
		if lobby.Player1.Score >= lobby.TargetScore || lobby.Player2.Score >= lobby.TargetScore {
			lobby.Status = "finished"
			var winnerPlayer *Player
			var loserPlayer *Player

			if lobby.Player2.Score >= lobby.TargetScore && lobby.Player2.Score > lobby.Player1.Score {
				winnerPlayer = lobby.Player2
				loserPlayer = lobby.Player1
			} else {
				winnerPlayer = lobby.Player1
				loserPlayer = lobby.Player2
			}

			winner := "player1" // Default winner
			if winnerPlayer == lobby.Player2 {
				winner = "player2" // Update if Player2 wins
			}

			log.Printf("[RollDice] Игра достигла цели. Победитель: %s (%s)",
				winner, winnerPlayer.FirstName)

			winAmount := lobby.BetAmount * 2   
			temp :=  ((lobby.BetAmount * 2) * 0.1 )  // Удвоенная ставка
			winAmountt := lobby.BetAmount - temp
			loseAmount := lobby.BetAmount         // Ставка проигравшего
			winAmountWithFee := winAmount * 0.9 

			ctx, cancel := s.withDBTimeout()
			defer cancel()

			log.Printf("[RollDice] Обновление балансов: Winner=%s, Loser=%s, WinAmount=%.2f, LoseAmount=%.2f",
				winnerPlayer.Wallet, loserPlayer.Wallet, winAmountt, loseAmount)
			err := s.userRepo.UpdateBalances(ctx, winnerPlayer.Wallet, loserPlayer.Wallet,
				lobby.TokenType, winAmountt, loseAmount)
			if err != nil {
				log.Printf("[RollDice] Ошибка обновления балансов: %v", err)
				s.safeWriteJSON(player1Conn, map[string]interface{}{
					"action":  "error",
					"message": "Ошибка обновления балансов",
				})
				s.safeWriteJSON(player2Conn, map[string]interface{}{
					"action":  "error",
					"message": "Ошибка обновления балансов",
				})
				s.lobbiesMu.Unlock()
				return
			}

			// Реферальная награда
			referralReward := lobby.BetAmount * 2 * 0.1
			referralService := referralServices.NewReferralService(s.userRepo)
			err = referralService.DistributeReferralReward(ctx, winnerPlayer.Wallet, referralReward, lobby.TokenType)
			if err != nil {
				log.Printf("[RollDice] Ошибка реферальной награды: %v", err)
				s.safeWriteJSON(winnerPlayer.Conn, map[string]interface{}{
					"action":  "error",
					"message": "Ошибка распределения реферальной награды",
				})
			}

			// Начисление очков
			err = s.userRepo.AddPointsForBet(ctx, winnerPlayer.Wallet, lobby.TokenType, lobby.BetAmount, true, "pvp")
			if err != nil {
				log.Printf("[RollDice] Ошибка начисления очков победителю: %v", err)
			}
			err = s.userRepo.AddPointsForBet(ctx, loserPlayer.Wallet, lobby.TokenType, lobby.BetAmount, false, "pvp")
			if err != nil {
				log.Printf("[RollDice] Ошибка начисления очков проигравшему: %v", err)
			}

			// ---- Исправление: отдельно считаем player1Earnings, player2Earnings ----
			var p1Earnings, p2Earnings float64
			if winnerPlayer == lobby.Player1 {
				p1Earnings = winAmountWithFee
				p2Earnings = -lobby.BetAmount
			} else {
				p1Earnings = -lobby.BetAmount
				p2Earnings = winAmountWithFee
			}

			// Сохраняем запись об игре
			errSave := s.gameService.SaveGame(
				ctx,
				lobby.Player1.FirstName,
				lobby.Player2.FirstName,
				lobby.Player1.Score,
				lobby.Player2.Score,
				winnerPlayer.FirstName, // Имя победителя
				p1Earnings,             // player1Earnings
				p2Earnings,             // player2Earnings
				lobby.TokenType,
				lobby.BetAmount,
				lobby.Player1.Wallet,
				lobby.Player2.Wallet,
			)
			if errSave != nil {
				log.Printf("[RollDice] Ошибка сохранения игры: %v", errSave)
			}

			// Удаляем лобби
			delete(s.lobbies, lobbyID)
			s.lobbiesMu.Unlock()

			// Рассылаем game_over с именем победителя
			gameOverMessage := map[string]interface{}{
				"action":      "game_over",
				"winner":      winner, // Имя победителя
				"winner_name": winnerPlayer.FirstName,
			}
			s.safeWriteJSON(player1Conn, gameOverMessage)
			s.safeWriteJSON(player2Conn, gameOverMessage)

			s.BroadcastLobbyList()
			log.Printf("[RollDice] Игра завершена. Победитель: %s", winner)
			return
		}

		// Если никто не достиг TargetScore, начинаем новый раунд
		log.Printf("[RollDice] Начало нового раунда: %d", lobby.CurrentRound+1)
		lobby.CurrentRound++
		lobby.RoundRolls = make(map[string]int)
	}

	// Передача хода следующему игроку
	lobby.CurrentTurn = getNextTurn(lobby)
	turnChangeMessage := map[string]interface{}{
		"action":       "turn_change",
		"current_turn": lobby.CurrentTurn,
	}
	s.lobbiesMu.Unlock()

	s.safeWriteJSON(player1Conn, turnChangeMessage)
	s.safeWriteJSON(player2Conn, turnChangeMessage)
	log.Printf("[RollDice] Следующий ход: %s", lobby.CurrentTurn)
}

// =======================================
// ConfirmReady
// =======================================
func (s *DicePVPGameService) ConfirmReady(player *Player, lobbyID string) error {
	s.lobbiesMu.Lock()
	lobby, exists := s.lobbies[lobbyID]
	if !exists || lobby.Status != "waiting" {
		s.lobbiesMu.Unlock()
		log.Printf("[ConfirmReady] Лобби не найдено или игра уже началась: %s", lobbyID)
		return fmt.Errorf("лобби не найдено или игра уже началась")
	}

	if lobby.Player1 == player {
		if lobby.ReadyPlayer1 {
			s.lobbiesMu.Unlock()
			log.Printf("[ConfirmReady] Игрок %s уже подтвердил готовность", player.FirstName)
			return fmt.Errorf("вы уже подтвердили свою готовность")
		}
		lobby.ReadyPlayer1 = true
	} else if lobby.Player2 == player {
		if lobby.ReadyPlayer2 {
			s.lobbiesMu.Unlock()
			log.Printf("[ConfirmReady] Игрок %s уже подтвердил готовность", player.FirstName)
			return fmt.Errorf("вы уже подтвердили свою готовность")
		}
		lobby.ReadyPlayer2 = true
	} else {
		s.lobbiesMu.Unlock()
		log.Printf("[ConfirmReady] Игрок %s не участвует в лобби %s", player.FirstName, lobbyID)
		return fmt.Errorf("вы не участвуете в этом лобби")
	}

	// Если оба готовы — стартуем игру
	if lobby.ReadyPlayer1 && lobby.ReadyPlayer2 {
		lobby.Status = "in_progress"
		s.lobbiesMu.Unlock()

		log.Println("[ConfirmReady] Оба игрока подтвердили готовность. Начало игры.")

		startMessagePlayer1 := map[string]interface{}{
			"action":        "game_start",
			"message":       "Игра начинается! Первый ход за Игроком 1.",
			"current_turn":  "player1",
			"player_id":     "player1",
			"player_name":   lobby.Player1.FirstName,
			"lobby_id":      lobby.ID,
			"target_score":  lobby.TargetScore,
			"current_round": lobby.CurrentRound,
			"token_type":    lobby.TokenType,
			"bet_amount":    lobby.BetAmount,
			"player1_id":    lobby.Player1.ID,        // Добавлено: ID Player1
			"player2_id":    lobby.Player2.ID,        // Добавлено: ID Player2
			"player1_name":  lobby.Player1.FirstName, // Добавлено: Имя Player1
			"player2_name":  lobby.Player2.FirstName, // Добавлено: Имя Player2
		}
		startMessagePlayer2 := map[string]interface{}{
			"action":        "game_start",
			"message":       "Игра начинается! Первый ход за Игроком 1.",
			"current_turn":  "player1",
			"player_id":     "player2",
			"player_name":   lobby.Player2.FirstName,
			"lobby_id":      lobby.ID,
			"target_score":  lobby.TargetScore,
			"current_round": lobby.CurrentRound,
			"token_type":    lobby.TokenType,
			"bet_amount":    lobby.BetAmount,
			"player1_id":    lobby.Player1.ID,        // Добавлено: ID Player1
			"player2_id":    lobby.Player2.ID,        // Добавлено: ID Player2
			"player1_name":  lobby.Player1.FirstName, // Добавлено: Имя Player1
			"player2_name":  lobby.Player2.FirstName, // Добавлено: Имя Player2
		}

		s.safeWriteJSON(lobby.Player1.Conn, startMessagePlayer1)
		s.safeWriteJSON(lobby.Player2.Conn, startMessagePlayer2)
		log.Printf("[ConfirmReady] Игра началась в лобби %s", lobbyID)
	} else {
		s.lobbiesMu.Unlock()
		log.Printf("[ConfirmReady] Игрок %s подтвердил готовность. Ожидаем второго игрока", player.FirstName)
	}

	return nil
}

// =======================================
// TerminateGame: исправленный вызов SaveGame
// =======================================
func (s *DicePVPGameService) TerminateGame(player *Player, lobbyID string, winner string) error {
	log.Printf("[TerminateGame] Досрочное завершение игры. PlayerID=%s, FirstName=%s, LobbyID=%s, Winner=%s",
		player.ID, player.FirstName, lobbyID, winner)

	s.lobbiesMu.Lock()
	defer s.lobbiesMu.Unlock()

	// Проверяем существование лобби
	lobby, exists := s.lobbies[lobbyID]
	if !exists || lobby.Status != "in_progress" {
		log.Printf("[TerminateGame] Лобби %s не найдено или игра уже завершена", lobbyID)
		return fmt.Errorf("лобби не найдено или игра уже завершена")
	}

	// Проверяем, является ли игрок участником лобби
	if player != lobby.Player1 && player != lobby.Player2 {
		log.Printf("[TerminateGame] Игрок %s (%s) не является участником лобби %s",
			player.ID, player.FirstName, lobbyID)
		return fmt.Errorf("вы не являетесь участником этого лобби")
	}

	// Определяем победителя и проигравшего
	var winnerPlayer, loserPlayer *Player
	if winner == "player1" {
		winnerPlayer = lobby.Player1
		loserPlayer = lobby.Player2
	} else if winner == "player2" {
		winnerPlayer = lobby.Player2
		loserPlayer = lobby.Player1
	} else {
		log.Printf("[TerminateGame] Неверный идентификатор победителя: %s", winner)
		return fmt.Errorf("неверный идентификатор победителя: %s", winner)
	}

	// Завершаем игру
	lobby.Status = "finished"
	log.Printf("[TerminateGame] Игра в лобби %s завершена. Победитель: %s (%s)", lobbyID, winner, winnerPlayer.FirstName)

	// Вычисляем выигрыш и проигрыш
	winAmount := lobby.BetAmount * 2 * 1.9
	loseAmount := lobby.BetAmount

	// Обновляем балансы игроков
	ctx, cancel := s.withDBTimeout()
	defer cancel()

	err := s.userRepo.UpdateBalances(ctx, winnerPlayer.Wallet, loserPlayer.Wallet, lobby.TokenType, winAmount/2, loseAmount)
	if err != nil {
		log.Printf("[TerminateGame] Ошибка обновления балансов: %v", err)
		return fmt.Errorf("не удалось обновить балансы игроков")
	}

	// Начисление реферальной награды
	referralReward := lobby.BetAmount * 0.1
	referralService := referralServices.NewReferralService(s.userRepo)
	err = referralService.DistributeReferralReward(ctx, winnerPlayer.Wallet, referralReward, lobby.TokenType)
	if err != nil {
		log.Printf("[TerminateGame] Ошибка начисления реферальной награды: %v", err)
	}

	// Начисление очков игрокам
	err = s.userRepo.AddPointsForBet(ctx, winnerPlayer.Wallet, lobby.TokenType, lobby.BetAmount, true, "pvp")
	if err != nil {
		log.Printf("[TerminateGame] Ошибка начисления очков победителю: %v", err)
	}
	err = s.userRepo.AddPointsForBet(ctx, loserPlayer.Wallet, lobby.TokenType, lobby.BetAmount, false, "pvp")
	if err != nil {
		log.Printf("[TerminateGame] Ошибка начисления очков проигравшему: %v", err)
	}

	// Сохранение записи об игре
	var p1Earnings, p2Earnings float64
	if winnerPlayer == lobby.Player1 {
		p1Earnings = winAmount
		p2Earnings = 0
	} else {
		p1Earnings = 0
		p2Earnings = winAmount
	}

	err = s.gameService.SaveGame(
		ctx,
		lobby.Player1.FirstName,
		lobby.Player2.FirstName,
		lobby.Player1.Score,
		lobby.Player2.Score,
		winnerPlayer.FirstName, // Имя победителя
		p1Earnings,
		p2Earnings,
		lobby.TokenType,
		lobby.BetAmount,
		lobby.Player1.Wallet,
		lobby.Player2.Wallet,
	)
	if err != nil {
		log.Printf("[TerminateGame] Ошибка сохранения игры: %v", err)
	}

	// Уведомляем игроков о завершении игры
	var winnerKey string
	if winnerPlayer == lobby.Player1 {
		winnerKey = "player1"
	} else {
		winnerKey = "player2"
	}

	gameOverMessage := map[string]interface{}{
		"action":      "game_over",
		"winner":      winnerKey,              // Передаем player1 или player2
		"winner_name": winnerPlayer.FirstName, // Имя победителя
	}
	s.safeWriteJSON(lobby.Player1.Conn, gameOverMessage)
	s.safeWriteJSON(lobby.Player2.Conn, gameOverMessage)

	// Удаляем лобби
	delete(s.lobbies, lobbyID)
	log.Printf("[TerminateGame] Лобби %s удалено после завершения игры", lobbyID)

	return nil
}

// =======================================
// Список лобби
// =======================================
func (s *DicePVPGameService) BroadcastLobbyList() {
	log.Println("[BroadcastLobbyList] Начало трансляции списка лобби всем клиентам")
	s.lobbiesMu.Lock()
	var availableLobbies []map[string]interface{}
	for id, lobby := range s.lobbies {
		if lobby.Status == "waiting" {
			availableLobbies = append(availableLobbies, map[string]interface{}{
				"lobby_id":     id,
				"creator_name": lobby.Player1.FirstName,
				"target_score": lobby.TargetScore,
				"token_type":   lobby.TokenType,
				"bet_amount":   lobby.BetAmount,
			})
		}
	}
	s.lobbiesMu.Unlock()

	message := map[string]interface{}{
		"action":  "lobby_list",
		"lobbies": availableLobbies,
	}

	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	for client := range s.clients {
		err := s.safeWriteJSON(client, message)
		if err != nil {
			log.Printf("[BroadcastLobbyList] Ошибка при отправке списка лобби клиенту %v: %v",
				client.RemoteAddr(), err)
			client.Close()
			delete(s.clients, client)
		}
	}

	log.Println("[BroadcastLobbyList] Трансляция списка лобби завершена")
}

func (s *DicePVPGameService) sendLobbyList(conn *websocket.Conn) {
	log.Println("[sendLobbyList] Начало отправки списка лобби клиенту")
	s.lobbiesMu.Lock()
	var availableLobbies []map[string]interface{}
	for id, lobby := range s.lobbies {
		if lobby.Status == "waiting" {
			availableLobbies = append(availableLobbies, map[string]interface{}{
				"lobby_id":     id,
				"creator_name": lobby.Player1.FirstName,
				"target_score": lobby.TargetScore,
				"token_type":   lobby.TokenType,
				"bet_amount":   lobby.BetAmount,
			})
		}
	}
	s.lobbiesMu.Unlock()

	message := map[string]interface{}{
		"action":  "lobby_list",
		"lobbies": availableLobbies,
	}

	err := s.safeWriteJSON(conn, message)
	if err != nil {
		log.Printf("[sendLobbyList] Ошибка при отправке списка лобби клиенту %v: %v",
			conn.RemoteAddr(), err)
	} else {
		log.Printf("[sendLobbyList] Список лобби успешно отправлен клиенту %v", conn.RemoteAddr())
	}
}

// =======================================
// History WebSocket Server (пример)
// =======================================
type WebSocketServer struct {
	clients   map[*websocket.Conn]bool
	clientsMu sync.Mutex
	upgrader  websocket.Upgrader
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (s *WebSocketServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	log.Println("[History] Инициализация нового History WebSocket-соединения")
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[History] Ошибка при обновлении WebSocket: %v", err)
		return
	}
	defer func() {
		conn.Close()
		log.Println("[History] History WebSocket-соединение закрыто")
	}()

	s.addClient(conn)
	defer s.removeClient(conn)

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("[History] Ошибка при чтении сообщения: %v", err)
			break
		}

		log.Printf("[History] Получено сообщение: %#v", msg)

		response := map[string]interface{}{
			"action":  "history_response",
			"message": "Получено ваше сообщение",
			"data":    msg,
		}

		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err = conn.WriteJSON(response)
		if err != nil {
			log.Printf("[History] Ошибка при отправке ответа: %v", err)
			break
		}
	}
}

func (s *WebSocketServer) addClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.clients[conn] = true
	log.Printf("[History] Новый клиент подключён: %v", conn.RemoteAddr())
}

func (s *WebSocketServer) removeClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	delete(s.clients, conn)
	log.Printf("[History] Клиент отключён: %v", conn.RemoteAddr())
}

// =======================================
// Utility Functions
// =======================================

// getInt безопасно извлекает целочисленное значение из сообщения
func getInt(message map[string]interface{}, key string, defaultValue int) (int, bool) {
	if val, exists := message[key]; exists {
		if floatVal, ok := val.(float64); ok {
			return int(floatVal), true
		}
	}
	return defaultValue, false
}

// getFloat64 безопасно извлекает float64 значение из сообщения
func getFloat64(message map[string]interface{}, key string, defaultValue float64) (float64, bool) {
	if val, exists := message[key]; exists {
		if floatVal, ok := val.(float64); ok {
			return floatVal, true
		}
	}
	return defaultValue, false
}

func getNextTurn(lobby *Lobby) string {
	if lobby.CurrentTurn == "player1" {
		return "player2"
	}
	return "player1"
}

func generateLobbyID() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func generatePlayerID() string {
	return fmt.Sprintf("player_%d", rand.Intn(1000000))
}

// Вспомогательная функция для контекста с таймаутом
func (s *DicePVPGameService) withDBTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Second)
}
