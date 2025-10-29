package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type ThreeDRecord struct {
	Date   string `json:"date"`
	Result string `json:"result"`
}

func main() {
	// Get database path from environment or use default
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./burma2d.db"
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Read JSON file
	jsonData, err := ioutil.ReadFile("../import-3d-data.json")
	if err != nil {
		log.Fatal("Failed to read JSON file:", err)
	}

	// Parse JSON
	var records []ThreeDRecord
	err = json.Unmarshal(jsonData, &records)
	if err != nil {
		log.Fatal("Failed to parse JSON:", err)
	}

	// Insert records
	insertQuery := `
		INSERT OR REPLACE INTO threed 
		(date, result, created_at, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	successCount := 0

	for _, record := range records {
		_, err := db.Exec(insertQuery,
			record.Date,
			record.Result,
		)

		if err != nil {
			log.Printf("⚠️ Error inserting record for %s: %v\n", record.Date, err)
		} else {
			successCount++
			fmt.Printf("✅ Inserted: %s -> %s\n", record.Date, record.Result)
		}
	}

	fmt.Printf("\n✅ Import completed!\n")
	fmt.Printf("   - Successfully inserted: %d records\n", successCount)
	fmt.Printf("   - Total processed: %d records\n", len(records))
}
