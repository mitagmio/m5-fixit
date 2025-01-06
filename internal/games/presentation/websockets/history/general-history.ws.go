package history

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type WebSocketServer struct {
	upgrader  websocket.Upgrader
	clients   map[*websocket.Conn]bool
	broadcast chan interface{}
}

// NewWebSocketServer создает новый экземпляр WebSocketServer
func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan interface{}), // Канал для отправки сообщений клиентам
	}
}

// HandleConnection обрабатывает входящее WebSocket соединение
func (ws *WebSocketServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка при установке WebSocket соединения:", err)
		return
	}
	defer conn.Close()

	// Добавляем клиента в список активных соединений
	ws.clients[conn] = true
	log.Println("Новое WebSocket соединение")

	// Обработка сообщений от клиента
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Ошибка при чтении сообщения:", err)
			delete(ws.clients, conn)
			break
		}
	}
}

// Broadcast отправляет сообщение всем подключённым клиентам
func (ws *WebSocketServer) Broadcast(message interface{}) {
	for client := range ws.clients {
		err := client.WriteJSON(message) // Отправка сообщения в формате JSON
		if err != nil {
			log.Println("Ошибка при отправке сообщения:", err)
			client.Close()
			delete(ws.clients, client)
		}
	}
}
