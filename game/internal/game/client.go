package game

import (
	"encoding/json"
	"log"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/models"
	"github.com/gorilla/websocket"
)

// WritePump пересылает сообщения из Go-канала client.Send в сеть (WebSocket)
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for message := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
}

// ReadPump читает команды из WebSocket и передает их в комнату
func (c *Client) ReadPump(room *Room) {
	defer func() {
		room.Leave(c)
		c.Conn.Close()
	}()

	for {
		_, p, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var action models.ActionMessage
		if err := json.Unmarshal(p, &action); err != nil {
			log.Printf("Ошибка декодирования команды: %v", err)
			continue
		}

		// Передаём обработку действия в комнату
		room.HandleAction(c, action)
	}
}
