package twodhistory

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// TwoDHistory represents a single lottery history record
type TwoDHistory struct {
	ID          int       `json:"history_id,omitempty" db:"id"`
	Date        string    `json:"draw_date" db:"date"`
	Set1200     string    `json:"noon_set" db:"set1200"`
	Value1200   string    `json:"noon_value" db:"value1200"`
	Result1200  string    `json:"noon_result" db:"result1200"`
	Set430      string    `json:"evening_set" db:"set430"`
	Value430    string    `json:"evening_value" db:"value430"`
	Result430   string    `json:"evening_result" db:"result430"`
	Modern930   string    `json:"morning_modern" db:"modern930"`
	Internet930 string    `json:"morning_internet" db:"internet930"`
	Modern200   string    `json:"afternoon_modern" db:"modern200"`
	Internet200 string    `json:"afternoon_internet" db:"internet200"`
	CreatedAt   time.Time `json:"created_date,omitempty" db:"created_at"`
}

var db *sql.DB

// LotteryData represents incoming lottery data (to avoid circular dependency)
type LotteryData struct {
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

// InitDB initializes the database connection
func InitDB(dbPath string) error {
	var err error

	log.Printf("📂 Opening database file: %s", dbPath)

	// Open SQLite database (will create file if it doesn't exist)
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create table if not exists
	if err = createTable(); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	log.Println("✅ Database connected and table created successfully")
	return nil
}

// createTable creates the twodhistory table if it doesn't exist
func createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS twodhistory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL UNIQUE,
		set1200 TEXT,
		value1200 TEXT,
		result1200 TEXT,
		set430 TEXT,
		value430 TEXT,
		result430 TEXT,
		modern930 TEXT,
		internet930 TEXT,
		modern200 TEXT,
		internet200 TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_twodhistory_date ON twodhistory(date DESC);
	`

	_, err := db.Exec(query)
	return err
}

// InsertHistory inserts a new history record if the date doesn't exist
func InsertHistory(history *TwoDHistory) error {
	// Check if date already exists
	exists, err := DateExists(history.Date)
	if err != nil {
		return err
	}

	if exists {
		log.Printf("⚠️  History for date %s already exists, skipping insert", history.Date)
		return nil
	}

	query := `
	INSERT INTO twodhistory (
		date, set1200, value1200, result1200,
		set430, value430, result430,
		modern930, internet930, modern200, internet200
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(query,
		history.Date,
		history.Set1200,
		history.Value1200,
		history.Result1200,
		history.Set430,
		history.Value430,
		history.Result430,
		history.Modern930,
		history.Internet930,
		history.Modern200,
		history.Internet200,
	)

	if err != nil {
		return fmt.Errorf("failed to insert history: %w", err)
	}

	log.Printf("✅ Inserted history for date: %s", history.Date)
	return nil
}

// InsertFromLotteryData inserts history from LotteryData struct
func InsertFromLotteryData(data *LotteryData) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	history := &TwoDHistory{
		Date:        data.Date,
		Set1200:     data.Set1200,
		Value1200:   data.Value1200,
		Result1200:  data.Result1200,
		Set430:      data.Set430,
		Value430:    data.Value430,
		Result430:   data.Result430,
		Modern930:   data.Modern930,
		Internet930: data.Internet930,
		Modern200:   data.Modern200,
		Internet200: data.Internet200,
	}

	return InsertHistory(history)
}

// DateExists checks if a history record for the given date already exists
func DateExists(date string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM twodhistory WHERE date = ?"
	err := db.QueryRow(query, date).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check date existence: %w", err)
	}
	return count > 0, nil
}

// GetAllHistory retrieves all history records ordered by date DESC
func GetAllHistory() ([]TwoDHistory, error) {
	query := `
	SELECT id, date, set1200, value1200, result1200,
	       set430, value430, result430,
	       modern930, internet930, modern200, internet200,
	       created_at
	FROM twodhistory
	ORDER BY date DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var histories []TwoDHistory
	for rows.Next() {
		var h TwoDHistory
		err := rows.Scan(
			&h.ID, &h.Date, &h.Set1200, &h.Value1200, &h.Result1200,
			&h.Set430, &h.Value430, &h.Result430,
			&h.Modern930, &h.Internet930, &h.Modern200, &h.Internet200,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		histories = append(histories, h)
	}

	return histories, nil
}

// GetHistoryHandler is the Gin handler for GET /api/twodhistory
func GetHistoryHandler(c *gin.Context) {
	histories, err := GetAllHistory()
	if err != nil {
		log.Printf("❌ Error fetching history: %v", err)
		c.JSON(500, gin.H{"error": "Failed to fetch history"})
		return
	}

	// Return empty array instead of null when no data
	if histories == nil || len(histories) == 0 {
		c.JSON(200, []TwoDHistory{})
		return
	}

	c.JSON(200, histories)
}

// CheckAndInsertHandler is the Gin handler for POST /api/twodhistory/check
// It checks if the date exists and inserts if not
func CheckAndInsertHandler(c *gin.Context) {
	var history TwoDHistory

	if err := c.BindJSON(&history); err != nil {
		log.Printf("❌ Error binding JSON: %v", err)
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	// Insert history (will skip if date already exists)
	if err := InsertHistory(&history); err != nil {
		log.Printf("❌ Error inserting history: %v", err)
		c.JSON(500, gin.H{"error": "Failed to insert history"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "History checked/inserted successfully",
		"date":    history.Date,
	})
}

// CloseDB closes the database connection
func CloseDB() {
	if db != nil {
		db.Close()
		log.Println("Database connection closed")
	}
}

// GetDB returns the database connection
func GetDB() *sql.DB {
	return db
}
