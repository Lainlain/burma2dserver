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

type TwoDRecord struct {
	Date        string `json:"date"`
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
	jsonData, err := ioutil.ReadFile("../import-2d-data.json")
	if err != nil {
		log.Fatal("Failed to read JSON file:", err)
	}

	// Parse JSON
	var records []TwoDRecord
	err = json.Unmarshal(jsonData, &records)
	if err != nil {
		log.Fatal("Failed to parse JSON:", err)
	}

	// Insert records
	insertQuery := `
		INSERT OR REPLACE INTO twodhistory 
		(date, set1200, value1200, result1200, set430, value430, result430, modern930, internet930, modern200, internet200)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	successCount := 0
	skipCount := 0

	for _, record := range records {
		// Skip records with missing data (----.-- or ---)
		if record.Set1200 == "----.--" || record.Result1200 == "--" {
			skipCount++
			continue
		}

		_, err := db.Exec(insertQuery,
			record.Date,
			record.Set1200,
			record.Value1200,
			record.Result1200,
			record.Set430,
			record.Value430,
			record.Result430,
			record.Modern930,
			record.Internet930,
			record.Modern200,
			record.Internet200,
		)

		if err != nil {
			log.Printf("⚠️ Error inserting record for %s: %v\n", record.Date, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("\n✅ Import completed!\n")
	fmt.Printf("   - Successfully inserted: %d records\n", successCount)
	fmt.Printf("   - Skipped (missing data): %d records\n", skipCount)
	fmt.Printf("   - Total processed: %d records\n", len(records))
}
