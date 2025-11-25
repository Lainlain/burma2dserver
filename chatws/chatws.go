package chatws

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/api/idtoken"
)

var db *sql.DB

// Myanmar timezone (Yangon - GMT+6:30)
var myanmarLocation *time.Location

// Firebase OAuth Client ID
var googleClientID string

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (configure in production)
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebSocket client management
type WSClient struct {
	UserID   string
	Username string
	PhotoURL string
	Conn     *websocket.Conn
	Send     chan []byte
}

var (
	clients      = make(map[*WSClient]bool)
	clientsMutex sync.RWMutex
	broadcast    = make(chan []byte, 256)
)

// Message represents a chat message
type Message struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	PhotoURL  string    `json:"photo_url"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// WSEvent types for WebSocket communication
type WSEvent struct {
	Type string      `json:"type"` // "message", "online_count", "user_joined", "user_left"
	Data interface{} `json:"data"`
}

// AuthRequest for initial WebSocket authentication
type AuthRequest struct {
	IDToken  string `json:"id_token"`
	Email    string `json:"email"`
	Username string `json:"username"`
	PhotoURL string `json:"photo_url"`
}

// Initialize database connection and timezone
func InitDB(database *sql.DB) error {
	db = database

	// Set Myanmar timezone
	var err error
	myanmarLocation, err = time.LoadLocation("Asia/Yangon")
	if err != nil {
		log.Printf("⚠️ Warning: Could not load Myanmar timezone, using UTC+6:30 offset: %v", err)
		myanmarLocation = time.FixedZone("Myanmar Time", 6*3600+30*60)
	}

	// Create tables if they don't exist
	createTables()

	// Start broadcast goroutine
	go handleBroadcast()

	log.Println("✅ WebSocket Chat initialized")
	return nil
}

// Set Google OAuth Client ID
func SetGoogleClientID(clientID string) {
	googleClientID = clientID
	log.Printf("✅ Google OAuth Client ID set for WebSocket chat: %s", clientID)
}

// Create necessary database tables
func createTables() {
	// Users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS chatws_users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			photo_url TEXT,
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_online BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("❌ Error creating chatws_users table: %v", err)
		return
	}

	// Messages table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chatws_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			photo_url TEXT,
			message TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES chatws_users(id)
		)
	`)
	if err != nil {
		log.Printf("❌ Error creating chatws_messages table: %v", err)
		return
	}

	// Blocked users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chatws_blocked_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			blocker_id TEXT NOT NULL,
			blocked_id TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(blocker_id, blocked_id),
			FOREIGN KEY (blocker_id) REFERENCES chatws_users(id),
			FOREIGN KEY (blocked_id) REFERENCES chatws_users(id)
		)
	`)
	if err != nil {
		log.Printf("❌ Error creating chatws_blocked_users table: %v", err)
	}

	log.Println("✅ WebSocket chat tables created/verified")
}

// WebSocket handler - main endpoint
func HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}

	// Wait for authentication message
	client, err := authenticateClient(conn)
	if err != nil {
		log.Printf("❌ WebSocket authentication failed: %v", err)
		conn.WriteJSON(map[string]string{"error": "Authentication failed"})
		conn.Close()
		return
	}

	// Register client
	clientsMutex.Lock()
	clients[client] = true
	clientsMutex.Unlock()

	log.Printf("✅ WebSocket client connected: %s (%s)", client.Username, client.UserID)

	// Update user online status
	updateUserOnlineStatus(client.UserID, true)

	// Notify others that user joined
	broadcastUserJoined(client)

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// Authenticate WebSocket client
func authenticateClient(conn *websocket.Conn) (*WSClient, error) {
	// Set read deadline for authentication
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Read authentication message
	var authReq AuthRequest
	err := conn.ReadJSON(&authReq)
	if err != nil {
		return nil, err
	}

	// Verify Google ID token
	payload, err := idtoken.Validate(context.Background(), authReq.IDToken, googleClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid ID token: %v", err)
	}

	userID := payload.Subject
	email := payload.Claims["email"].(string)

	// Create or update user in database
	_, err = db.Exec(`
		INSERT INTO chatws_users (id, email, username, photo_url, is_online, last_seen)
		VALUES (?, ?, ?, ?, TRUE, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			username = excluded.username,
			photo_url = excluded.photo_url,
			is_online = TRUE,
			last_seen = CURRENT_TIMESTAMP
	`, userID, email, authReq.Username, authReq.PhotoURL)

	if err != nil {
		log.Printf("⚠️ Error updating user: %v", err)
	}

	// Remove read deadline
	conn.SetReadDeadline(time.Time{})

	// Create client
	client := &WSClient{
		UserID:   userID,
		Username: authReq.Username,
		PhotoURL: authReq.PhotoURL,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	return client, nil
}

// Read pump - reads messages from WebSocket
func (c *WSClient) readPump() {
	defer func() {
		c.disconnect()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg map[string]interface{}
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error: %v", err)
			}
			break
		}

		// Handle different message types
		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "message":
			c.handleChatMessage(msg)
		case "ping":
			c.Send <- []byte(`{"type":"pong"}`)
		}
	}
}

// Write pump - writes messages to WebSocket
func (c *WSClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
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

// Handle incoming chat message
func (c *WSClient) handleChatMessage(msg map[string]interface{}) {
	messageText, ok := msg["message"].(string)
	if !ok || messageText == "" {
		return
	}

	// Save message to database
	result, err := db.Exec(`
		INSERT INTO chatws_messages (user_id, username, photo_url, message, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, c.UserID, c.Username, c.PhotoURL, messageText, time.Now().In(myanmarLocation))

	if err != nil {
		log.Printf("❌ Error saving message: %v", err)
		return
	}

	messageID, _ := result.LastInsertId()

	// Create message object
	chatMessage := Message{
		ID:        messageID,
		UserID:    c.UserID,
		Username:  c.Username,
		PhotoURL:  c.PhotoURL,
		Message:   messageText,
		CreatedAt: time.Now().In(myanmarLocation),
	}

	// Broadcast to all clients
	event := WSEvent{
		Type: "message",
		Data: chatMessage,
	}

	eventJSON, _ := json.Marshal(event)
	broadcast <- eventJSON

	log.Printf("💬 Message from %s: %s", c.Username, messageText)
}

// Disconnect client
func (c *WSClient) disconnect() {
	clientsMutex.Lock()
	if _, ok := clients[c]; ok {
		delete(clients, c)
		close(c.Send)
	}
	clientsMutex.Unlock()

	// Update user online status
	updateUserOnlineStatus(c.UserID, false)

	// Notify others that user left
	broadcastUserLeft(c)

	log.Printf("👋 WebSocket client disconnected: %s", c.Username)
}

// Broadcast goroutine
func handleBroadcast() {
	for {
		message := <-broadcast
		clientsMutex.RLock()
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(clients, client)
			}
		}
		clientsMutex.RUnlock()
	}
}

// Broadcast user joined event
func broadcastUserJoined(client *WSClient) {
	event := WSEvent{
		Type: "user_joined",
		Data: map[string]interface{}{
			"user_id":  client.UserID,
			"username": client.Username,
			"count":    getOnlineCount(),
		},
	}

	eventJSON, _ := json.Marshal(event)
	broadcast <- eventJSON
}

// Broadcast user left event
func broadcastUserLeft(client *WSClient) {
	event := WSEvent{
		Type: "user_left",
		Data: map[string]interface{}{
			"user_id":  client.UserID,
			"username": client.Username,
			"count":    getOnlineCount(),
		},
	}

	eventJSON, _ := json.Marshal(event)
	broadcast <- eventJSON
}

// Update user online status in database
func updateUserOnlineStatus(userID string, isOnline bool) {
	_, err := db.Exec(`
		UPDATE chatws_users 
		SET is_online = ?, last_seen = CURRENT_TIMESTAMP
		WHERE id = ?
	`, isOnline, userID)

	if err != nil {
		log.Printf("❌ Error updating user status: %v", err)
	}
}

// Get online user count
func getOnlineCount() int {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	return len(clients)
}

// HTTP endpoint to get recent messages
func GetRecentMessagesHandler(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")

	rows, err := db.Query(`
		SELECT id, user_id, username, photo_url, message, created_at
		FROM chatws_messages
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.Username, &msg.PhotoURL, &msg.Message, &msg.CreatedAt)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	// Reverse to chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, messages)
}

// HTTP endpoint to get online count
func GetOnlineCountHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"count": getOnlineCount(),
	})
}
