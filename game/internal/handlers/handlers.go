package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/economy"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/game"
)

func HandleVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		renderVerificationPage(w, false, "В ссылке отсутствует магический токен!")
		return
	}

	authServerURL := fmt.Sprintf("%s/verify?token=%s", os.Getenv("AUTH_SERVER_URL"), token)

	resp, err := http.Get(authServerURL)
	if err != nil {
		renderVerificationPage(w, false, "Таверна временно недоступна. Ошибка связи с сервером авторизации.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		renderVerificationPage(w, true, "Почта успешно подтверждена! Теперь вы можете войти в таверну.")
	} else {
		renderVerificationPage(w, false, "Свиток недействителен. Возможно, срок действия ссылки истёк.")
	}
}

func renderVerificationPage(w http.ResponseWriter, isSuccess bool, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	title := "Ошибка!"
	color := "#e74c3c"
	if isSuccess {
		title = "Успех!"
		color = "#2ecc71"
	}

	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="ru">
		<head>
			<meta charset="UTF-8">
			<title>Подтверждение почты</title>
			<style>
				body {
					background-color: #1a252f;
					color: #ecf0f1;
					font-family: sans-serif;
					display: flex;
					flex-direction: column;
					align-items: center;
					justify-content: center;
					height: 100vh;
					margin: 0;
					text-align: center;
				}
				.card {
					background: rgba(20, 20, 20, 0.85);
					border: 3px solid #915508;
					border-radius: 12px;
					padding: 40px;
					box-shadow: 0 8px 25px rgba(0, 0, 0, 0.7);
				}
				h1 { color: %s; }
				a {
					display: inline-block;
					margin-top: 20px;
					background-color: #915508;
					color: white;
					text-decoration: none;
					padding: 10px 20px;
					border: 2px solid #f1c40f;
					border-radius: 6px;
					font-weight: bold;
				}
				a:hover { background-color: #b56c0a; }
			</style>
		</head>
		<body>
			<div class="card">
				<h1>%s</h1>
				<p>%s</p>
				<a href="/">Вернуться ко входу</a>
			</div>
		</body>
		</html>
	`, color, title, message)

	if !isSuccess {
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Write([]byte(html))
}

func LeaderboardHandler(store economy.PlayerStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profiles := store.GetAllProfiles()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profiles)
	}
}

type CreateRoomRequest struct {
	BetAmount int `json:"bet_amount"`
}

// GET /api/rooms
func GetRoomsHandler(lobby *game.Lobby) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rooms := lobby.GetAvailableRooms()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rooms)
	}
}

// POST /api/rooms
func CreateRoomHandler(lobby *game.Lobby) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := r.Context().Value("username").(string)
		if !ok || username == "" {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		var req CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Неверный запрос", http.StatusBadRequest)
			return
		}

		room, err := lobby.CreateRoom(username, req.BetAmount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusPaymentRequired)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"room_id": room.ID})
	}
}

// POST /api/rooms/join?id=xxx
func JoinRoomHandler(lobby *game.Lobby) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := r.Context().Value("username").(string)
		if !ok || username == "" {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		roomID := r.URL.Query().Get("id")

		room, err := lobby.JoinRoom(roomID, username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusPaymentRequired)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "room_id": room.ID})
	}
}

type UpdateAvatarRequest struct {
	Avatar string `json:"avatar"`
}

// POST /api/user/avatar
func UpdateAvatarHandler(store economy.PlayerStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}

		username, ok := r.Context().Value("username").(string)
		if !ok || username == "" {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		var req UpdateAvatarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Некорректный JSON", http.StatusBadRequest)
			return
		}

		if req.Avatar == "" {
			http.Error(w, "Имя аватара не может быть пустым", http.StatusBadRequest)
			return
		}

		// Сохранение обновлений в базе данных через PlayerStore
		if err := store.UpdateAvatar(username, req.Avatar); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"avatar": req.Avatar,
		})
	}
}

// GET /api/user/profile
func GetProfileHandler(store economy.PlayerStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := r.Context().Value("username").(string)
		if !ok || username == "" {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		profile := store.GetProfile(username) // Внутри проверится еженедельное начисление

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}
}
