package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID            string
	UserID        uuid.UUID
	Conn          *websocket.Conn
	Send          chan []byte
	Hub           *WebSocketHub
	SubscribedIDs map[uuid.UUID]bool // Quote IDs this client is subscribed to
	mu            sync.RWMutex
}

// WebSocketHub maintains active WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan *models.QuoteStatusNotification
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan *models.QuoteStatusNotification, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			fmt.Printf("Client registered: %s\n", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			fmt.Printf("Client unregistered: %s\n", client.ID)

		case notification := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// Check if client is subscribed to this quote
				if client.IsSubscribed(notification.QuoteID) {
					select {
					case client.Send <- h.encodeNotification(notification):
					default:
						// Client's send buffer is full, close connection
						h.mu.RUnlock()
						h.unregister <- client
						h.mu.RLock()
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// RegisterClient registers a client with the hub
func (h *WebSocketHub) RegisterClient(client *WebSocketClient) {
	h.register <- client
}

// UnregisterClient unregisters a client from the hub
func (h *WebSocketHub) UnregisterClient(client *WebSocketClient) {
	h.unregister <- client
}

// encodeNotification encodes a notification to JSON
func (h *WebSocketHub) encodeNotification(notification *models.QuoteStatusNotification) []byte {
	data, err := json.Marshal(notification)
	if err != nil {
		fmt.Printf("Failed to encode notification: %v\n", err)
		return nil
	}
	return data
}

// Subscribe subscribes a client to quote updates
func (c *WebSocketClient) Subscribe(quoteID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SubscribedIDs[quoteID] = true
}

// Unsubscribe unsubscribes a client from quote updates
func (c *WebSocketClient) Unsubscribe(quoteID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.SubscribedIDs, quoteID)
}

// IsSubscribed checks if a client is subscribed to a quote
func (c *WebSocketClient) IsSubscribed(quoteID uuid.UUID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SubscribedIDs[quoteID]
}

// ReadPump reads messages from the WebSocket connection
func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket error: %v\n", err)
			}
			break
		}

		// Handle client messages (subscribe/unsubscribe)
		var msg struct {
			Action  string    `json:"action"` // "subscribe" or "unsubscribe"
			QuoteID uuid.UUID `json:"quote_id"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			fmt.Printf("Failed to unmarshal message: %v\n", err)
			continue
		}

		switch msg.Action {
		case "subscribe":
			c.Subscribe(msg.QuoteID)
			fmt.Printf("Client %s subscribed to quote %s\n", c.ID, msg.QuoteID)
		case "unsubscribe":
			c.Unsubscribe(msg.QuoteID)
			fmt.Printf("Client %s unsubscribed from quote %s\n", c.ID, msg.QuoteID)
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *WebSocketClient) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// WebSocketNotificationService implements NotificationService using WebSockets
type WebSocketNotificationService struct {
	hub *WebSocketHub
}

// NewWebSocketNotificationService creates a new WebSocket notification service
func NewWebSocketNotificationService(hub *WebSocketHub) *WebSocketNotificationService {
	return &WebSocketNotificationService{
		hub: hub,
	}
}

// SendQuoteStatusNotification sends a quote status notification via WebSocket
func (s *WebSocketNotificationService) SendQuoteStatusNotification(ctx context.Context, notification *models.QuoteStatusNotification) error {
	select {
	case s.hub.broadcast <- notification:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("notification send timeout")
	}
}

// GetHub returns the WebSocket hub
func (s *WebSocketNotificationService) GetHub() *WebSocketHub {
	return s.hub
}
