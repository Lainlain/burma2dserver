package live

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LotteryDataInput represents incoming data with old JSON key format from API runner
type LotteryDataInput struct {
	Date        string `json:"date"`
	Live        string `json:"live"`
	Status      string `json:"status"`
	Set1200     string `json:"1200set"`
	Value1200   string `json:"1200value"`
	Result1200  string `json:"1200"`
	Set430      string `json:"430set"`
	Value430    string `json:"430value"`
	Result430   string `json:"430"`
	Modern930   string `json:"930modern"`
	Internet930 string `json:"930internet"`
	Modern200   string `json:"200modern"`
	Internet200 string `json:"200internet"`
	UpdateTime  string `json:"updatetime"`
}

// LotteryData represents the lottery information with new JSON key format for output
type LotteryData struct {
	Date        string `json:"draw_date"`
	Live        string `json:"live_number"`
	Status      string `json:"service_status"`
	Set1200     string `json:"noon_set"`
	Value1200   string `json:"noon_value"`
	Result1200  string `json:"noon_result"`
	Set430      string `json:"evening_set"`
	Value430    string `json:"evening_value"`
	Result430   string `json:"evening_result"`
	Modern930   string `json:"morning_modern"`
	Internet930 string `json:"morning_internet"`
	Modern200   string `json:"afternoon_modern"`
	Internet200 string `json:"afternoon_internet"`
	UpdateTime  string `json:"last_update"`
	ViewCount   int    `json:"active_viewers"`
}

// ToLotteryData converts LotteryDataInput to LotteryData
func (input *LotteryDataInput) ToLotteryData() *LotteryData {
	// Helper function to replace empty strings with "--"
	defaultVal := func(val string) string {
		if val == "" {
			return "--"
		}
		return val
	}

	defaultResult := func(val string) string {
		if val == "" {
			return "---"
		}
		return val
	}

	return &LotteryData{
		Date:        input.Date,
		Live:        defaultVal(input.Live),
		Status:      input.Status,
		Set1200:     defaultVal(input.Set1200),
		Value1200:   defaultVal(input.Value1200),
		Result1200:  defaultResult(input.Result1200),
		Set430:      defaultVal(input.Set430),
		Value430:    defaultVal(input.Value430),
		Result430:   defaultResult(input.Result430),
		Modern930:   defaultVal(input.Modern930),
		Internet930: defaultVal(input.Internet930),
		Modern200:   defaultVal(input.Modern200),
		Internet200: defaultVal(input.Internet200),
		UpdateTime:  input.UpdateTime,
		ViewCount:   0, // Will be set by server
	}
}

// HistoryInserter is a callback function type for inserting history
type HistoryInserter func(data *LotteryData) error

// Global state
var (
	currentData     *LotteryData
	dataMutex       sync.RWMutex
	clients         = make(map[chan string]bool)
	clientsMutex    sync.RWMutex
	historyInserter HistoryInserter
	lastCheckTime   time.Time
	
	// Performance optimization: Reuse JSON buffers
	jsonBufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	
	// Cached JSON string to avoid re-marshaling for every client
	cachedJSONMessage string
	cachedJSONMutex   sync.RWMutex
)

// SetHistoryInserter sets the callback function for history insertion
func SetHistoryInserter(inserter HistoryInserter) {
	historyInserter = inserter
	log.Println("✅ History inserter callback registered")
}

// Init initializes the live package with default data
func Init() {
	currentData = &LotteryData{
		Live:        "--",
		Status:      "Off",
		Set1200:     "--",
		Value1200:   "--",
		Result1200:  "---",
		Set430:      "--",
		Value430:    "--",
		Result430:   "---",
		Modern930:   "---",
		Internet930: "---",
		Modern200:   "---",
		Internet200: "---",
		UpdateTime:  time.Now().Format("15:04:05 02/01/2006"),
	}
	log.Println("✅ Live package initialized with default data")
}

// UpdateLotteryData handles POST requests to update lottery data
func UpdateLotteryData(c *gin.Context) {
	var inputData LotteryDataInput

	// Read and parse JSON body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to read request body"})
		return
	}

	if err := json.Unmarshal(body, &inputData); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}

	// Transform input data to output format
	newData := inputData.ToLotteryData()

	// Update current data
	dataMutex.Lock()
	currentData = newData
	dataMutex.Unlock()

	log.Printf("📊 Lottery data updated - Live: %s, Status: %s", newData.Live, newData.Status)

	// Check if we should insert to history database (16:30-16:35 GMT+6:30)
	checkAndInsertHistory(newData)

	// Broadcast to all SSE clients
	broadcastUpdate()

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "Data updated successfully",
		"data":    newData,
	})
}

// checkAndInsertHistory checks if current time is 16:30-16:35 GMT+6:30 and inserts to database
func checkAndInsertHistory(data *LotteryData) {
	if historyInserter == nil {
		return // No history inserter registered
	}

	// Get Myanmar time (GMT+6:30)
	loc, err := time.LoadLocation("Asia/Yangon")
	if err != nil {
		log.Printf("❌ Error loading timezone: %v", err)
		return
	}

	now := time.Now().In(loc)
	hour := now.Hour()
	minute := now.Minute()

	// Check if time is between 16:30 and 16:35
	if hour == 16 && minute >= 30 && minute < 35 {
		// Check if 430 result has real data (not "--")
		if data.Result430 == "--" || data.Result430 == "" {
			log.Printf("⏭️  Skipping insert - 430 result is not ready yet: %s", data.Result430)
			return
		}

		// Avoid duplicate checks within the same minute
		if time.Since(lastCheckTime) < time.Minute {
			return
		}
		lastCheckTime = now

		log.Printf("⏰ Time check: %02d:%02d - Within insert window (16:30-16:35)", hour, minute)
		log.Printf("📊 430 result is ready: %s - Attempting to insert history for date: %s", data.Result430, data.Date)

		// Call the history inserter callback
		if err := historyInserter(data); err != nil {
			log.Printf("❌ Error inserting history: %v", err)
		} else {
			log.Printf("✅ History checked/inserted for date: %s", data.Date)
		}
	}
}

// GetCurrentData returns the current lottery data
func GetCurrentData(c *gin.Context) {
	dataMutex.RLock()
	data := currentData
	dataMutex.RUnlock()

	c.JSON(200, gin.H{
		"status": "success",
		"data":   data,
	})
}

// StreamLotteryData handles SSE streaming for real-time updates
func StreamLotteryData(c *gin.Context) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create a client channel with larger buffer for high concurrency (50 instead of 10)
	clientChan := make(chan string, 50)

	// Register client
	clientsMutex.Lock()
	clients[clientChan] = true
	clientCount := len(clients)
	clientsMutex.Unlock()

	// Log less frequently at high concurrency (every 100 connections)
	if clientCount%100 == 0 || clientCount < 100 {
		log.Printf("📡 New SSE client connected (Total clients: %d)", clientCount)
	}

	// Send initial data immediately with current client count
	// Use cached JSON if available, or marshal new data
	cachedJSONMutex.RLock()
	initialMessage := cachedJSONMessage
	cachedJSONMutex.RUnlock()
	
	if initialMessage == "" {
		// No cached data, marshal fresh
		dataMutex.RLock()
		currentData.ViewCount = clientCount
		initialData, _ := json.Marshal(currentData)
		dataMutex.RUnlock()
		initialMessage = string(initialData)
	}

	c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", initialMessage)))
	c.Writer.Flush()

	// Listen for updates and client disconnect
	notify := c.Request.Context().Done()

	for {
		select {
		case <-notify:
			// Client disconnected
			clientsMutex.Lock()
			delete(clients, clientChan)
			remainingClients := len(clients)
			clientsMutex.Unlock()
			close(clientChan)
			
			// Log less frequently at high concurrency
			if remainingClients%100 == 0 || remainingClients < 100 {
				log.Printf("📴 SSE client disconnected (Remaining clients: %d)", remainingClients)
			}
			return
		case message := <-clientChan:
			// Send update to client
			c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", message)))
			c.Writer.Flush()
		}
	}
}

// broadcastUpdate sends updates to all connected SSE clients
// OPTIMIZED for 10,000+ concurrent connections
func broadcastUpdate() {
	// Step 1: Get client count first (quick lock)
	clientsMutex.RLock()
	clientCount := len(clients)
	clientsMutex.RUnlock()
	
	// Step 2: Marshal JSON once using buffer pool (no lock needed)
	buf := jsonBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	
	dataMutex.RLock()
	currentData.ViewCount = clientCount
	encoder := json.NewEncoder(buf)
	err := encoder.Encode(currentData)
	dataMutex.RUnlock()
	
	if err != nil {
		log.Printf("❌ Failed to marshal data: %v", err)
		jsonBufferPool.Put(buf)
		return
	}
	
	// Convert to string and cache it
	message := buf.String()
	jsonBufferPool.Put(buf)
	
	// Cache the JSON message for new connections
	cachedJSONMutex.Lock()
	cachedJSONMessage = message
	cachedJSONMutex.Unlock()
	
	// Step 3: Broadcast to all clients (minimize lock time)
	clientsMutex.RLock()
	
	// Count skipped clients
	skippedCount := 0
	sentCount := 0
	
	for clientChan := range clients {
		select {
		case clientChan <- message:
			sentCount++
		default:
			// Channel is full, skip this client (prevents blocking)
			skippedCount++
		}
	}
	
	clientsMutex.RUnlock()
	
	// Log only if there are issues or every 10th broadcast
	if skippedCount > 0 {
		log.Printf("⚠️  Broadcast: %d sent, %d skipped (full buffers) out of %d clients", 
			sentCount, skippedCount, clientCount)
	} else if clientCount%1000 == 0 || clientCount < 1000 {
		log.Printf("📤 Broadcast to %d clients (all sent)", clientCount)
	}
}
