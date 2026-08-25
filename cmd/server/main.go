package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/economy"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/game"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/handlers"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/ws"
	"github.com/joho/godotenv"
)

var economyStore = economy.NewStore("profiles.json")
var lobby = game.NewLobby(economyStore)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка при загрузке файла .env")
	}

	http.Handle("/sounds/", http.StripPrefix("/sounds/", http.FileServer(http.Dir("./sounds"))))
	http.Handle("/pictures/", http.StripPrefix("/pictures/", http.FileServer(http.Dir("./pictures"))))
	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(w, r, lobby)
	})
	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleVerify(w, r)
	})
	http.HandleFunc("/api/profile", handlers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Если фронтенд и бэкенд на разных портах, разрешаем CORS
		// w.Header().Set("Access-Control-Allow-Origin", "*")
		// w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		// Извлекаем имя пользователя из токена
		username := r.Context().Value("username").(string)
		// Получаем профиль игрока из хранилища
		profile := economyStore.GetProfile(username)

		// Отправляем данные обратно в формате JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))

	http.HandleFunc("/api/leaderboard", handlers.LeaderboardHandler(economyStore))
	http.HandleFunc("/api/rooms", handlers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetRoomsHandler(lobby)(w, r)
		case http.MethodPost:
			handlers.CreateRoomHandler(lobby)(w, r)
		default:
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/api/rooms/join", handlers.AuthMiddleware(handlers.JoinRoomHandler(lobby)))

	port := ":8081"
	log.Printf("🚀 Сервер NeeKoobaMeemo запущен на http://localhost%s", port)
	log.Printf("📁 Статические файлы: ./web")
	log.Printf("🔌 WebSocket: ws://localhost%s/ws", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("❌ Не удалось запустить сервер: %v", err)
	}
}
