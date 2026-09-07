package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/config"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/db"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/economy"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/game"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/handlers"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/ws"
)

func main() {
	// 1. Загрузка конфигурации из пакета config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка при загрузке конфигурации: %v", err)
	}

	// 2. Инициализация базы данных с DSN из конфигуратора
	database, err := db.InitDB(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("Не удалось запустить БД: %v", err)
	}
	defer database.Close()

	// Внедрение зависимости в стор
	playerStore := economy.NewDBStore(database)
	var lobby = game.NewLobby(playerStore)

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
		profile := playerStore.GetProfile(username)

		// Отправляем данные обратно в формате JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))

	http.HandleFunc("/api/leaderboard", handlers.LeaderboardHandler(playerStore))
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
	http.HandleFunc("/api/user/avatar", handlers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.UpdateAvatarHandler(lobby.Store)(w, r)
		default:
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	}))

	port := ":8081"
	log.Printf("🚀 Сервер NeeKoobaMeemo запущен на http://localhost%s", port)
	log.Printf("📁 Статические файлы: ./web")
	log.Printf("🔌 WebSocket: ws://localhost%s/ws", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("❌ Не удалось запустить сервер: %v", err)
	}
}
