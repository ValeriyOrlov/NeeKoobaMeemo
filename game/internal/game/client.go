package game

import (
	"encoding/json"
	"log"
	"time"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/models"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10 // Отправлять пинг каждые ~54 секунды
)

// WritePump пересылает сообщения из Go-канала client.Send в сеть (WebSocket)
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			// Автоматическая отправка Ping-кадра клиенту
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump читает команды из WebSocket и передает их в комнату
func (c *Client) ReadPump(room *Room) {
	defer func() {
		room.Leave(c)
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(1024)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	// Сбрасываем таймер таймаута при каждом получении Pong от браузера
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

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
