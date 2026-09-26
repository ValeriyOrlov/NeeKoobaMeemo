package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/config"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/db"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/economy"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/game"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/handlers"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/ws"
)

func resolveDir(defaultPath, fallbackPath string) string {
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}
	return fallbackPath
}

func main() {
	// 1. Загрузка конфигурации из пакета config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка при загрузке конфигурации: %v", err)
	}
	secretBytes := []byte(cfg.JWTSecret)

	// 2. Инициализация базы данных с DSN из конфигуратора
	database, err := db.InitDB(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("Не удалось запустить БД: %v", err)
	}
	defer database.Close()

	// Внедрение зависимости в стор
	playerStore := economy.NewDBStore(database)
	var lobby = game.NewLobby(playerStore)

	statusHandler := handlers.HandleCheckPlayerStatus(lobby)

	picturesDir := resolveDir("./pictures", "./game/pictures")
	soundsDir := resolveDir("./sounds", "./game/sounds")
	webDir := resolveDir("./web", "./game/web")

	http.Handle("/sounds/", http.StripPrefix("/sounds/", http.FileServer(http.Dir(soundsDir))))
	http.Handle("/pictures/", http.StripPrefix("/pictures/", http.FileServer(http.Dir(picturesDir))))
	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(w, r, lobby, secretBytes)
	})
	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleVerify(w, r)
	})

	authMiddleware := handlers.AuthMiddleware(secretBytes)
	http.HandleFunc("/api/profile", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем имя пользователя из токена
		username := r.Context().Value("username").(string)
		// Получаем профиль игрока из хранилища
		profile := playerStore.GetProfile(username)

		// Отправляем данные обратно в формате JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))

	http.HandleFunc("/api/leaderboard", handlers.LeaderboardHandler(playerStore))
	http.HandleFunc("/api/rooms", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetRoomsHandler(lobby)(w, r)
		case http.MethodPost:
			handlers.CreateRoomHandler(lobby)(w, r)
		default:
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/api/rooms/join", authMiddleware(handlers.JoinRoomHandler(lobby)))
	http.HandleFunc("/api/user/avatar", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.UpdateAvatarHandler(lobby.Store)(w, r)
		default:
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	}))
	http.HandleFunc("/internal/player-status", handlers.InternalAuthMiddleware(statusHandler))

	http.HandleFunc("/env.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")

		// Преобразуем GAME_SERVER_URL (http:// -> ws://, https:// -> wss://) для локального WS
		localWS := strings.Replace(cfg.GameServerURL, "https://", "wss://", 1)
		localWS = strings.Replace(localWS, "http://", "ws://", 1)

		// Генерируем JS, который умеет определять локальный запуск (порт 8081) и работу через Nginx
		jsContent := fmt.Sprintf(`(function () {
  const { protocol, hostname, port, host } = window.location;
  const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:';
  const isLocalDev = port === '8081';

  window.ENV = {
    GAME_SERVER_URL: isLocalDev ? "%s" : protocol + "//" + host,
    AUTH_SERVER_URL: isLocalDev ? "%s" : protocol + "//" + host + "/api/auth",
    WS_URL: isLocalDev ? "%s/ws" : wsProtocol + "//" + host + "/ws"
  };
})();`, cfg.GameServerURL, cfg.AuthServerURL, localWS)

		w.Write([]byte(jsContent))
	})

	port := ":8081"
	log.Printf("🚀 Сервер NeeKoobaMeemo запущен на http://localhost%s", port)
	log.Printf("📁 Статические файлы: ./web")
	log.Printf("🔌 WebSocket: ws://localhost%s/ws", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("❌ Не удалось запустить сервер: %v", err)
	}
}
