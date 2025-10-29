package threed

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type ThreeDResult struct {
	ID        int       `json:"id"`
	Date      string    `json:"date"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var db *sql.DB

// InitDB initializes the database connection
func InitDB(database *sql.DB) {
	db = database
	createTable()
}

// createTable creates the threed table if it doesn't exist
func createTable() {
	query := `
		CREATE TABLE IF NOT EXISTS threed (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date DATE NOT NULL UNIQUE,
			result TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_threed_date ON threed(date DESC);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Error creating threed table: %v", err)
	}
}

// GetAllResults fetches all 3D results ordered by date DESC
func GetAllResults(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, date, result, created_at, updated_at 
		FROM threed 
		ORDER BY date DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []ThreeDResult
	for rows.Next() {
		var result ThreeDResult
		var date time.Time
		err := rows.Scan(&result.ID, &date, &result.Result, &result.CreatedAt, &result.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		result.Date = date.Format("2006-01-02")
		results = append(results, result)
	}

	c.JSON(http.StatusOK, results)
}

// CreateResult creates a new 3D result
func CreateResult(c *gin.Context) {
	var input struct {
		Date   string `json:"date"`
		Result string `json:"result"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Validate result (must be 3 digits)
	if len(input.Result) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Result must be 3 digits"})
		return
	}

	// Parse and validate date
	_, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	query := `
		INSERT INTO threed (date, result, created_at, updated_at) 
		VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, date, result, created_at, updated_at
	`

	var result ThreeDResult
	var date time.Time
	err = db.QueryRow(query, input.Date, input.Result).Scan(
		&result.ID, &date, &result.Result, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Result for this date already exists or database error"})
		return
	}

	result.Date = date.Format("2006-01-02")
	c.JSON(http.StatusCreated, result)
}

// UpdateResult updates an existing 3D result
func UpdateResult(c *gin.Context) {
	var input struct {
		ID     int    `json:"id"`
		Date   string `json:"date"`
		Result string `json:"result"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Validate result
	if len(input.Result) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Result must be 3 digits"})
		return
	}

	query := `
		UPDATE threed 
		SET result = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $2
		RETURNING id, date, result, created_at, updated_at
	`

	var result ThreeDResult
	var date time.Time
	err := db.QueryRow(query, input.Result, input.ID).Scan(
		&result.ID, &date, &result.Result, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Result not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	result.Date = date.Format("2006-01-02")
	c.JSON(http.StatusOK, result)
}

// DeleteResult deletes a 3D result
func DeleteResult(c *gin.Context) {
	var input struct {
		ID int `json:"id"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	result, err := db.Exec("DELETE FROM threed WHERE id = $1", input.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Result not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
