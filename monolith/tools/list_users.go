
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Use a default connection string for local development
		dbURL = "postgres://user:password@localhost:5432/codejudgedb?sslmode=disable"
		fmt.Println("INFO: DATABASE_URL environment variable not set. Using default value.")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("ERROR: Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ERROR: Failed to connect to the database. Is it running? %v", err)
	}


rows, err := db.Query("SELECT id, username, role FROM users ORDER BY id")
	if err != nil {
		log.Fatalf("ERROR: Failed to query users: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n--- Users in Database ---")
	fmt.Printf("%-5s | %-20s | %-10s\n", "ID", "Username", "Role")
	fmt.Println("-----------------------------------------")

	count := 0
	for rows.Next() {
		var id int
		var username, role string
		if err := rows.Scan(&id, &username, &role); err != nil {
			log.Fatalf("ERROR: Failed to scan row: %v", err)
		}
		fmt.Printf("%-5d | %-20s | %-10s\n", id, username, role)
		count++
	}
	
	if count == 0 {
		fmt.Println("No users found in the database.")
	}
	fmt.Println("-----------------------------------------")
}
