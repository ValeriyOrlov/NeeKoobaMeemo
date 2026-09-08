package ws

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/game"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/websocket"
)

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

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))

	// 3. Парсим и валидируем токен
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

	// 4. Извлекаем данные (claims)
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

	roomID := r.URL.Query().Get("room_id")
	var room *game.Room

	if roomID != "" {
		// Классический сценарий: Игрок явно создаёт комнату или присоединяется к ней
		room, err = lobby.GetRoom(roomID)
		if err != nil {
			http.Error(w, "Комната не найдена", http.StatusNotFound)
			return
		}
	} else {
		// Сценарий перезагрузки (F5): room_id нет, ищем игрока в отключившихся
		room = lobby.FindRoomByDisconnectedPlayer(username)
		if room == nil {
			http.Error(w, "Вы не находитесь в активной игре", http.StatusBadRequest)
			return
		}
	}

	// 5. Обновляем соединение до WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("handleWebSocket connection error: %v", err)
		return
	}

	// 6. Создаем клиента, используя надёжное имя из JWT
	client := &game.Client{
		Conn:       conn,
		PlayerName: username,
		Send:       make(chan []byte, 256),
	}

	// 7. Регистрируем клиента в комнате и запускаем pumps
	room.AddClient(client)
	// Запускаем горутину записи в WebSocket
	go client.WritePump()
	// Чтение запускаем в текущей горутине
	go client.ReadPump(room)
}
