package main

import (
	"log"
	"net/http"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/game"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/models"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var lobby = game.NewLobby()

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("handleWebSocket connection error: %v", err)
		return
	}

	// Создаем нового клиента
	client := &game.Client{
		Conn:       conn,
		PlayerName: r.URL.Query().Get("name"),
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
func main() {
	http.Handle("/pictures/", http.StripPrefix("/pictures/", http.FileServer(http.Dir("./pictures"))))
	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.HandleFunc("/ws", handleWebSocket)
	port := ":8080"
	log.Printf("🚀 Сервер NeeKoobaMeemo запущен на http://localhost%s", port)
	log.Printf("📁 Статические файлы: ./web")
	log.Printf("🔌 WebSocket: ws://localhost%s/ws", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("❌ Не удалось запустить сервер: %v", err)
	}
}
