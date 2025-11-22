package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
)

var db *sql.DB

// Myanmar timezone (Yangon - GMT+6:30)
var myanmarLocation *time.Location

// Firebase OAuth Client ID (replace with your actual client ID)
var googleClientID string

// SSE clients management
type SSEClient struct {
	UserID   string
	Username string
	PhotoURL string
	Channel  chan []byte
}

var (
	clients      = make(map[string]*SSEClient)
	clientsMutex sync.RWMutex
)

// User represents a chat user (from Google OAuth)
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	PhotoURL  string    `json:"photo_url"`
	LastSeen  time.Time `json:"last_seen"`
	IsOnline  bool      `json:"is_online"`
	CreatedAt time.Time `json:"created_at"`
}

// Message represents a chat message
type Message struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	PhotoURL  string    `json:"photo_url"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// BlockedUser represents a block relationship
type BlockedUser struct {
	ID        int64     `json:"id"`
	BlockerID string    `json:"blocker_id"`
	BlockedID string    `json:"blocked_id"`
	CreatedAt time.Time `json:"created_at"`
}

// OnlineUser represents an online user with details
type OnlineUser struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	PhotoURL string `json:"photo_url"`
}

// SSE Event types
type SSEEvent struct {
	Type string      `json:"type"` // "message", "online", "offline", "count"
	Data interface{} `json:"data"`
}

// OnlineStatus represents online user count and list
type OnlineStatus struct {
	Count int          `json:"count"`
	Users []OnlineUser `json:"users"`
}

// InitDB initializes the database
func InitDB(database *sql.DB) error {
	db = database

	// Load Myanmar timezone (Asia/Yangon - GMT+6:30)
	var err error
	myanmarLocation, err = time.LoadLocation("Asia/Yangon")
	if err != nil {
		log.Printf("⚠️  Failed to load Asia/Yangon timezone, using fixed offset GMT+6:30: %v", err)
		// Fallback: Create fixed offset for Myanmar (GMT+6:30 = 6.5 hours = 23400 seconds)
		myanmarLocation = time.FixedZone("Myanmar", 6*3600+30*60)
	}
	log.Printf("✅ Chat timezone set to Myanmar (GMT+6:30)")

	return createTables()
}

// SetGoogleClientID sets the Google OAuth client ID for token verification
func SetGoogleClientID(clientID string) {
	googleClientID = clientID
	log.Printf("✅ Google OAuth Client ID configured for chat")
}

func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS chat_users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			photo_url TEXT,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_online BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			photo_url TEXT,
			message TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES chat_users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS chat_blocks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			blocker_id TEXT NOT NULL,
			blocked_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(blocker_id, blocked_id),
			FOREIGN KEY (blocker_id) REFERENCES chat_users(id),
			FOREIGN KEY (blocked_id) REFERENCES chat_users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS chat_banned_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL,
			banned_by TEXT DEFAULT 'admin',
			reason TEXT DEFAULT 'Violation of community guidelines',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES chat_users(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created ON chat_messages(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_users_online ON chat_users(is_online)`,
		`CREATE INDEX IF NOT EXISTS idx_banned_users ON chat_banned_users(user_id)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	log.Println("✅ Chat tables created successfully")
	return nil
}

// RegisterRoutes registers chat endpoints
func RegisterRoutes(router *gin.Engine) {
	chat := router.Group("/api/burma2d/chat")
	{
		// Authentication & User Management
		chat.POST("/auth/google", googleAuthHandler)
		chat.GET("/users/online", getOnlineUsersHandler)

		// Messaging
		chat.POST("/messages", sendMessageHandler)
		chat.GET("/messages", getMessagesHandler)

		// Blocking
		chat.POST("/block", blockUserHandler)
		chat.POST("/unblock", unblockUserHandler)
		chat.GET("/blocked", getBlockedUsersHandler)

		// Admin: Ban Management
		chat.POST("/admin/ban", banUserHandler)
		chat.POST("/admin/unban", unbanUserHandler)
		chat.GET("/admin/banned", getBannedUsersHandler)
		chat.GET("/admin/messages", getAllMessagesHandler)

		// SSE Stream
		chat.GET("/stream", sseStreamHandler)
	}
}

// googleAuthHandler handles Google OAuth login with Firebase token verification
func googleAuthHandler(c *gin.Context) {
	var req struct {
		IDToken  string `json:"id_token" binding:"required"`
		Email    string `json:"email"`
		Username string `json:"username"`
		PhotoURL string `json:"photo_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify Google ID token
	var userID, email, username, photoURL string

	if googleClientID != "" {
		// Verify token with Google
		ctx := context.Background()
		payload, err := idtoken.Validate(ctx, req.IDToken, googleClientID)
		if err != nil {
			log.Printf("⚠️  Token validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid ID token"})
			return
		}

		// Extract user info from verified token
		userID = payload.Claims["email"].(string)
		email = payload.Claims["email"].(string)

		// Get username from token or request
		if name, ok := payload.Claims["name"].(string); ok && name != "" {
			username = name
		} else if req.Username != "" {
			username = req.Username
		} else {
			username = email
		}

		// Get photo URL from token or request
		if picture, ok := payload.Claims["picture"].(string); ok && picture != "" {
			photoURL = picture
		} else if req.PhotoURL != "" {
			photoURL = req.PhotoURL
		}

		log.Printf("✅ Token verified for user: %s", email)
	} else {
		// Fallback: Development mode without verification
		log.Println("⚠️  Running without Google OAuth verification (development mode)")
		userID = req.Email
		email = req.Email
		username = req.Username
		photoURL = req.PhotoURL

		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email required in development mode"})
			return
		}
	}

	// Insert or update user with verified data
	_, err := db.Exec(`
		INSERT INTO chat_users (id, email, username, photo_url, is_online)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT(id) DO UPDATE SET
			username = excluded.username,
			photo_url = excluded.photo_url,
			is_online = 1,
			last_seen = CURRENT_TIMESTAMP
	`, userID, email, username, photoURL)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
		return
	}

	// Get user data
	var user User
	err = db.QueryRow(`
		SELECT id, email, username, photo_url, last_seen, is_online, created_at
		FROM chat_users WHERE id = ?
	`, userID).Scan(&user.ID, &user.Email, &user.Username, &user.PhotoURL,
		&user.LastSeen, &user.IsOnline, &user.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Broadcast online status
	broadcastOnlineStatus()

	c.JSON(http.StatusOK, gin.H{
		"user_id":   user.ID,
		"username":  user.Username,
		"photo_url": user.PhotoURL,
		"message":   "Authentication successful",
	})
}

// sendMessageHandler handles sending a message
func sendMessageHandler(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user is banned
	if isUserBanned(req.UserID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":  "You have been banned from the chat",
			"banned": true,
		})
		return
	}

	// Get user info
	var username, photoURL string
	err := db.QueryRow(`
		SELECT username, photo_url FROM chat_users WHERE id = ?
	`, req.UserID).Scan(&username, &photoURL)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Insert message
	result, err := db.Exec(`
		INSERT INTO chat_messages (user_id, username, photo_url, message)
		VALUES (?, ?, ?, ?)
	`, req.UserID, username, photoURL, req.Message)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	messageID, _ := result.LastInsertId()

	// Create message object with Myanmar time (GMT+6:30)
	message := Message{
		ID:        messageID,
		UserID:    req.UserID,
		Username:  username,
		PhotoURL:  photoURL,
		Message:   req.Message,
		CreatedAt: time.Now().In(myanmarLocation), // Always Myanmar Yangon time
	}

	// Broadcast to all connected clients
	broadcastMessage(message, req.UserID)

	// Return response matching Android app expectations
	c.JSON(http.StatusOK, gin.H{
		"message_id": messageID,
		"message":    req.Message,
	})
}

// getMessagesHandler gets recent messages
func getMessagesHandler(c *gin.Context) {
	userID := c.Query("user_id")
	limit := c.DefaultQuery("limit", "30")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	// Get blocked users
	blockedIDs, err := getBlockedUserIDs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get blocked users"})
		return
	}

	// Build query to exclude blocked users
	query := `
		SELECT id, user_id, username, photo_url, message, created_at
		FROM chat_messages
		WHERE user_id NOT IN (?)
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := db.Query(query, blockedIDs, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.Username, &msg.PhotoURL,
			&msg.Message, &msg.CreatedAt)
		if err != nil {
			continue
		}
		// Convert to Myanmar timezone (GMT+6:30)
		msg.CreatedAt = msg.CreatedAt.In(myanmarLocation)
		messages = append(messages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"messages": messages,
	})
}

// blockUserHandler blocks a user
func blockUserHandler(c *gin.Context) {
	var req struct {
		BlockerID string `json:"blocker_id" binding:"required"`
		BlockedID string `json:"blocked_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`
		INSERT OR IGNORE INTO chat_blocks (blocker_id, blocked_id)
		VALUES (?, ?)
	`, req.BlockerID, req.BlockedID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// unblockUserHandler unblocks a user
func unblockUserHandler(c *gin.Context) {
	var req struct {
		BlockerID string `json:"blocker_id" binding:"required"`
		BlockedID string `json:"blocked_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`
		DELETE FROM chat_blocks
		WHERE blocker_id = ? AND blocked_id = ?
	`, req.BlockerID, req.BlockedID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// getBlockedUsersHandler gets blocked users
func getBlockedUsersHandler(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	rows, err := db.Query(`
		SELECT u.id, u.username, u.photo_url
		FROM chat_blocks b
		JOIN chat_users u ON b.blocked_id = u.id
		WHERE b.blocker_id = ?
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get blocked users"})
		return
	}
	defer rows.Close()

	var blocked []OnlineUser
	for rows.Next() {
		var user OnlineUser
		rows.Scan(&user.UserID, &user.Username, &user.PhotoURL)
		blocked = append(blocked, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"blocked": blocked,
	})
}

// getOnlineUsersHandler gets online users with username and photo
func getOnlineUsersHandler(c *gin.Context) {
	userID := c.Query("user_id")

	// Get blocked users to exclude
	blockedIDs, _ := getBlockedUserIDs(userID)

	rows, err := db.Query(`
		SELECT id, username, photo_url
		FROM chat_users
		WHERE is_online = 1 AND id NOT IN (?)
		ORDER BY username ASC
	`, blockedIDs)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get online users"})
		return
	}
	defer rows.Close()

	var online []OnlineUser
	for rows.Next() {
		var user OnlineUser
		rows.Scan(&user.UserID, &user.Username, &user.PhotoURL)
		online = append(online, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(online),
		"users":   online,
	})
}

// sseStreamHandler handles SSE connections
func sseStreamHandler(c *gin.Context) {
	userID := c.Query("user_id")
	username := c.Query("username")
	photoURL := c.Query("photo_url")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create client
	client := &SSEClient{
		UserID:   userID,
		Username: username,
		PhotoURL: photoURL,
		Channel:  make(chan []byte, 10),
	}

	// Register client
	clientsMutex.Lock()
	clients[userID] = client
	clientsMutex.Unlock()

	// Set user online
	db.Exec("UPDATE chat_users SET is_online = 1, last_seen = CURRENT_TIMESTAMP WHERE id = ?", userID)

	// Broadcast online status
	broadcastOnlineStatus()

	// Send initial connection message with online count
	onlineCount := getOnlineCount()
	event := SSEEvent{
		Type: "connected",
		Data: gin.H{
			"user_id":      userID,
			"online_count": onlineCount,
		},
	}
	sendSSE(c.Writer, event)

	// Create context with cancellation for proper cleanup
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Heartbeat ticker to keep connection alive (every 15 seconds)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Listen for messages
	for {
		select {
		case <-ctx.Done():
			// Client disconnected or context cancelled
			clientsMutex.Lock()
			delete(clients, userID)
			clientsMutex.Unlock()

			// Set user offline
			db.Exec("UPDATE chat_users SET is_online = 0, last_seen = CURRENT_TIMESTAMP WHERE id = ?", userID)

			// Broadcast offline status
			broadcastOnlineStatus()
			log.Printf("🔌 SSE client disconnected: %s", userID)
			return
		case <-ticker.C:
			// Send heartbeat to keep connection alive
			_, err := c.Writer.Write([]byte(": heartbeat\n\n"))
			if err != nil {
				log.Printf("❌ SSE heartbeat failed for %s: %v", userID, err)
				return
			}
			c.Writer.(http.Flusher).Flush()
		case msg := <-client.Channel:
			_, err := c.Writer.Write(msg)
			if err != nil {
				log.Printf("❌ SSE write failed for %s: %v", userID, err)
				return
			}
			c.Writer.(http.Flusher).Flush()
		}
	}
}

// Helper functions

func getBlockedUserIDs(userID string) (string, error) {
	if userID == "" {
		return "''", nil
	}

	rows, err := db.Query(`
		SELECT blocked_id FROM chat_blocks WHERE blocker_id = ?
	`, userID)
	if err != nil {
		return "''", err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, "'"+id+"'")
	}

	if len(ids) == 0 {
		return "''", nil
	}

	// Return ALL blocked IDs, not just the first one
	return strings.Join(ids, ","), nil
}

func broadcastMessage(message Message, senderID string) {
	log.Printf("💬💬💬 BROADCAST MESSAGE CALLED! 💬💬💬")
	log.Printf("📧 Message: %s", message.Message)
	log.Printf("👤 Sender: %s (ID: %s)", message.Username, senderID)
	
	event := SSEEvent{
		Type: "message",
		Data: message,
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("❌ Failed to marshal message event: %v", err)
		return
	}
	sseData := []byte(fmt.Sprintf("data: %s\n\n", data))
	
	log.Printf("📦 SSE Data: %s", string(sseData))

	clientsMutex.RLock()
	connectedClients := len(clients)
	log.Printf("👥 Connected SSE clients: %d", connectedClients)
	clientsMutex.RUnlock()

	if connectedClients == 0 {
		log.Printf("⚠️ No SSE clients connected - message not broadcast")
		return
	}

	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	sentCount := 0
	blockedCount := 0
	
	for userID, client := range clients {
		// Send to everyone including sender (so they see their own message)
		// But skip blocked users

		// Check if sender is blocked by this user
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM chat_blocks
			WHERE blocker_id = ? AND blocked_id = ?
		`, userID, senderID).Scan(&count)
		
		if err != nil {
			log.Printf("⚠️ Error checking block status for user %s: %v", userID, err)
		}

		if count == 0 {
			select {
			case client.Channel <- sseData:
				sentCount++
				log.Printf("✅ Sent message to client: %s (%s)", client.Username, userID)
			default:
				// Channel full, skip
				log.Printf("⚠️ Channel full for client: %s (%s)", client.Username, userID)
			}
		} else {
			blockedCount++
			log.Printf("🚫 Skipped blocked user: %s", userID)
		}
	}
	
	log.Printf("📊 Broadcast complete: Sent to %d clients, Blocked %d", sentCount, blockedCount)
}

func broadcastOnlineStatus() {
	// Get all online users
	rows, _ := db.Query(`
		SELECT id, username, photo_url
		FROM chat_users
		WHERE is_online = 1
		ORDER BY username ASC
	`)
	defer rows.Close()

	var online []OnlineUser
	for rows.Next() {
		var user OnlineUser
		rows.Scan(&user.UserID, &user.Username, &user.PhotoURL)
		online = append(online, user)
	}

	status := OnlineStatus{
		Count: len(online),
		Users: online,
	}

	event := SSEEvent{
		Type: "online",
		Data: status,
	}

	data, _ := json.Marshal(event)
	sseData := []byte(fmt.Sprintf("data: %s\n\n", data))

	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	for _, client := range clients {
		select {
		case client.Channel <- sseData:
		default:
		}
	}
}

func getOnlineCount() int {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM chat_users WHERE is_online = 1").Scan(&count)
	return count
}

func sendSSE(w http.ResponseWriter, event SSEEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
	w.(http.Flusher).Flush()
}

// ============================================
// Admin Ban Management Handlers
// ============================================

// banUserHandler bans a user and deletes all their messages
func banUserHandler(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		Reason   string `json:"reason"`
		BannedBy string `json:"banned_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default reason if not provided
	if req.Reason == "" {
		req.Reason = "Violation of community guidelines"
	}

	// Set default banned_by if not provided
	if req.BannedBy == "" {
		req.BannedBy = "admin"
	}

	// Get username for the user
	var username string
	err := db.QueryRow("SELECT username FROM chat_users WHERE id = ?", req.UserID).Scan(&username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Insert into banned_users table
	_, err = tx.Exec(`
		INSERT INTO chat_banned_users (user_id, username, banned_by, reason)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			banned_by = excluded.banned_by,
			reason = excluded.reason,
			created_at = CURRENT_TIMESTAMP
	`, req.UserID, username, req.BannedBy, req.Reason)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ban user"})
		return
	}

	// Delete all messages from this user
	result, err := tx.Exec("DELETE FROM chat_messages WHERE user_id = ?", req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user messages"})
		return
	}

	deletedCount, _ := result.RowsAffected()

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	log.Printf("✅ User banned: %s (%s) - Deleted %d messages - Reason: %s", username, req.UserID, deletedCount, req.Reason)

	c.JSON(http.StatusOK, gin.H{
		"message":          "User banned successfully",
		"user_id":          req.UserID,
		"username":         username,
		"deleted_messages": deletedCount,
		"reason":           req.Reason,
	})
}

// unbanUserHandler removes a user from the banned list
func unbanUserHandler(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec("DELETE FROM chat_banned_users WHERE user_id = ?", req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unban user"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found in banned list"})
		return
	}

	log.Printf("✅ User unbanned: %s", req.UserID)

	c.JSON(http.StatusOK, gin.H{
		"message": "User unbanned successfully",
		"user_id": req.UserID,
	})
}

// getAllMessagesHandler gets all messages for admin (no filtering)
func getAllMessagesHandler(c *gin.Context) {
	limit := c.DefaultQuery("limit", "100")

	rows, err := db.Query(`
		SELECT id, user_id, username, photo_url, message, created_at
		FROM chat_messages
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.Username, &msg.PhotoURL, &msg.Message, &msg.CreatedAt)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	if messages == nil {
		messages = []Message{}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"count":    len(messages),
	})
}

// getBannedUsersHandler returns list of all banned users
func getBannedUsersHandler(c *gin.Context) {
	rows, err := db.Query(`
		SELECT user_id, username, banned_by, reason, created_at
		FROM chat_banned_users
		ORDER BY created_at DESC
	`)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get banned users"})
		return
	}
	defer rows.Close()

	var bannedUsers []map[string]interface{}
	for rows.Next() {
		var userID, username, bannedBy, reason string
		var createdAt time.Time

		err := rows.Scan(&userID, &username, &bannedBy, &reason, &createdAt)
		if err != nil {
			continue
		}

		bannedUsers = append(bannedUsers, map[string]interface{}{
			"user_id":   userID,
			"username":  username,
			"banned_by": bannedBy,
			"reason":    reason,
			"banned_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"banned_users": bannedUsers,
		"count":        len(bannedUsers),
	})
}

// isUserBanned checks if a user is banned
func isUserBanned(userID string) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM chat_banned_users WHERE user_id = ?", userID).Scan(&count)
	return err == nil && count > 0
}
