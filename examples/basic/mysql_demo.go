package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/koustreak/DatRi/internal/database"
	"github.com/koustreak/DatRi/internal/database/mysql"
)

func main() {
	ctx := context.Background()

	// --- 1. Connect ---
	db := mysql.New(&database.Config{
		Host:     getEnv("MYSQL_HOST", "localhost"),
		Port:     3306,
		User:     getEnv("MYSQL_USER", "datri"),
		Password: getEnv("MYSQL_PASSWORD", "datri_secret"),
		Database: getEnv("MYSQL_DB", "datri_test"),
		MaxConns: 10,
		MinConns: 2,
	})

	fmt.Println("Connecting to MySQL...")
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close(ctx)
	fmt.Println("✅ Connected")

	// --- 2. Ping ---
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}
	fmt.Println("✅ Ping OK")

	// --- 3. Create table ---
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id         INT AUTO_INCREMENT PRIMARY KEY,
			name       VARCHAR(100) NOT NULL,
			email      VARCHAR(100) NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("create table: %v", err)
	}
	fmt.Println("✅ Table ready")

	// --- 4. Insert rows ---
	seeds := []struct{ name, email string }{
		{"Alice", "alice@example.com"},
		{"Bob", "bob@example.com"},
		{"Charlie", "charlie@example.com"},
	}

	for _, s := range seeds {
		_, err := db.Exec(ctx,
			"INSERT IGNORE INTO users (name, email) VALUES (?, ?)",
			s.name, s.email,
		)
		if err != nil {
			log.Fatalf("insert %s: %v", s.name, err)
		}
	}
	fmt.Println("✅ Rows seeded")

	// --- 5. Query all rows ---
	fmt.Println("\n📋 Users:")
	rows, err := db.Query(ctx, "SELECT id, name, email, created_at FROM users ORDER BY id")
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, email string
		var createdAt time.Time

		if err := rows.Scan(&id, &name, &email, &createdAt); err != nil {
			log.Fatalf("scan: %v", err)
		}
		fmt.Printf("  [%d] %-10s %-25s %s\n", id, name, email, createdAt.Format(time.RFC3339))
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows err: %v", err)
	}

	// --- 6. Query single row ---
	fmt.Println("\n🔍 Lookup by email:")
	var id int
	var name string
	err = db.QueryRow(ctx,
		"SELECT id, name FROM users WHERE email = ?",
		"alice@example.com",
	).Scan(&id, &name)
	if err != nil {
		log.Fatalf("query row: %v", err)
	}
	fmt.Printf("  Found: id=%d name=%s\n", id, name)

	// --- 7. Transaction ---
	fmt.Println("\n🔄 Transaction:")
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}

	_, err = tx.Exec(ctx,
		"INSERT IGNORE INTO users (name, email) VALUES (?, ?)",
		"Dave", "dave@example.com",
	)
	if err != nil {
		tx.Rollback(ctx)
		log.Fatalf("tx insert: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}
	fmt.Println("  ✅ Transaction committed")

	// --- 8. Error handling demo ---
	fmt.Println("\n⚠️  Error handling (duplicate email):")
	_, err = db.Exec(ctx,
		"INSERT INTO users (name, email) VALUES (?, ?)",
		"Duplicate", "alice@example.com", // already exists
	)
	if err != nil {
		var dbErr *database.DBError
		if isDBError(err, &dbErr) {
			fmt.Printf("  Kind=%v Message=%s\n", dbErr.Kind, dbErr.Message)
		}
	}

	fmt.Println("\n✅ MySQL demo complete")
}

func isDBError(err error, target **database.DBError) bool {
	if e, ok := err.(*database.DBError); ok {
		*target = e
		return true
	}
	return false
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
