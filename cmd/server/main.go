package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/economy"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/game"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/models"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/websocket"
)

var jwtSecret = []byte("secret")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWs(w http.ResponseWriter, r *http.Request, lobby *game.Lobby) {
	// 1. Получаем токен из URL параметров (ws://localhost:8080/ws?token=eyJ...)
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "Token is required", http.StatusUnauthorized)
		return
	}

	// 2. Парсим и валидируем токен
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// 3. Извлекаем данные (claims)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	// Достаем имя пользователя, которое генерируется в auth-сервисе
	username, ok := claims["username"].(string)
	if !ok {
		http.Error(w, "Username not found in token", http.StatusUnauthorized)
		return
	}

	// 4. Обновляем соединение до WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("handleWebSocket connection error: %v", err)
		return
	}

	// 5. Создаем клиента, используя надёжное имя из JWT
	client := &game.Client{
		Conn:       conn,
		PlayerName: username,
		Send:       make(chan []byte, 256),
	}

	if client.PlayerName == "" {
		client.PlayerName = "Аноним"
	}

	// Находим или создаем комнату
	room := lobby.JoinOrCreateRoom(client)

	// Запускаем горутину записи в WebSocket
	go client.WritePump()

	// Оповещаем комнату о подключении
	room.Broadcast(models.EventMessage{
		Type:    "SYSTEM",
		Message: client.PlayerName + " присоединился к игре!",
	})

	// Чтение запускаем в текущей горутине
	client.ReadPump(room)
}

var economyStore = economy.NewStore("profiles.json")
var lobby = game.NewLobby(economyStore, 50)

func main() {
	http.Handle("/pictures/", http.StripPrefix("/pictures/", http.FileServer(http.Dir("./pictures"))))
	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(w, r, lobby)
	})
	port := ":8081"
	log.Printf("🚀 Сервер NeeKoobaMeemo запущен на http://localhost%s", port)
	log.Printf("📁 Статические файлы: ./web")
	log.Printf("🔌 WebSocket: ws://localhost%s/ws", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("❌ Не удалось запустить сервер: %v", err)
	}
}
