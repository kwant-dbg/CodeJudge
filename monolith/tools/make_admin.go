package main

import (
	"database/sql"
	"fmt"
	"os"

	"codejudge/common/env"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run make_admin.go <username>")
		os.Exit(1)
	}
	username := os.Args[1]

	databaseURL := env.Get("DATABASE_URL", "")
	if databaseURL == "" {
		fmt.Println("DATABASE_URL not set")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Printf("Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	query := `UPDATE users SET role = 'admin' WHERE username = $1`
	result, err := db.Exec(query, username)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err)
		os.Exit(1)
	}


rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Failed to get rows affected: %v\n", err)
		os.Exit(1)
	}

	if rowsAffected == 0 {
		fmt.Printf("No user found with username: %s\n", username)
	} else {
		fmt.Printf("User '%s' has been granted admin privileges.\n", username)
	}
}
