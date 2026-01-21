package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/javaknight1/servicepro/backend/internal/services"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// WebSocketHandler handles WebSocket connections for real-time notifications
type WebSocketHandler struct {
	notificationService *services.WebSocketNotificationService
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(notificationService *services.WebSocketNotificationService) *WebSocketHandler {
	return &WebSocketHandler{
		notificationService: notificationService,
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade connection"})
		return
	}

	// Create new client
	client := &services.WebSocketClient{
		ID:            uuid.New().String(),
		UserID:        userID,
		Conn:          conn,
		Send:          make(chan []byte, 256),
		Hub:           h.notificationService.GetHub(),
		SubscribedIDs: make(map[uuid.UUID]bool),
	}

	// Register client with hub
	client.Hub.RegisterClient(client)

	// Start read and write pumps
	go client.WritePump()
	go client.ReadPump()
}

// HandleHealthCheck handles WebSocket health check endpoint
func (h *WebSocketHandler) HandleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "websocket",
	})
}
