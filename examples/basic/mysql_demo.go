package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/koustreak/DatRi/internal/database"
	"github.com/koustreak/DatRi/internal/database/mysql"
)

func main() {
	ctx := context.Background()
	dbName := getEnv("MYSQL_DB", "datri_test")

	// --- Connect ---
	db := mysql.New(&database.Config{
		Host:     getEnv("MYSQL_HOST", "localhost"),
		Port:     3306,
		User:     getEnv("MYSQL_USER", "datri"),
		Password: getEnv("MYSQL_PASSWORD", "datri_secret"),
		Database: dbName,
	})

	fmt.Println("Connecting to MySQL...")
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close(ctx)
	fmt.Println("✅ Connected")

	// --- Ping ---
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}
	fmt.Println("✅ Ping OK")

	// --- MySQL version ---
	var version string
	if err := db.QueryRow(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		log.Fatalf("version: %v", err)
	}
	fmt.Printf("✅ MySQL version: %s\n", version)

	// --- Introspect schema ---
	fmt.Printf("\n📦 Introspecting database: %s\n", dbName)
	introspector := mysql.NewIntrospector(db)

	tables, err := introspector.ListTables(ctx, dbName)
	if err != nil {
		log.Fatalf("list tables: %v", err)
	}

	fmt.Printf("📋 Tables found: %d\n", len(tables))
	if len(tables) == 0 {
		fmt.Println("  (no tables — create some in your DB first)")
		fmt.Println("\n💡 Example:")
		fmt.Println("  CREATE TABLE users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(100), email VARCHAR(100) UNIQUE);")
		fmt.Println("  CREATE TABLE posts (id INT AUTO_INCREMENT PRIMARY KEY, user_id INT, title VARCHAR(255), FOREIGN KEY (user_id) REFERENCES users(id));")
		return
	}

	// --- Inspect each table ---
	fmt.Println("\n🔍 Table Details:")
	for _, t := range tables {
		info, err := introspector.InspectTable(ctx, dbName, t)
		if err != nil {
			log.Printf("  inspect %s: %v", t, err)
			continue
		}

		fmt.Printf("\n  ┌─ %s\n", info.Name)
		for _, col := range info.Columns {
			fmt.Printf("  │  %-20s %-15s %s\n", col.Name, col.DataType, buildFlags(col))
		}
		fmt.Printf("  └─ (%d columns)\n", len(info.Columns))
	}

	// --- Foreign keys ---
	fmt.Println("\n🔗 Foreign Keys:")
	schema, err := database.InspectSchema(ctx, introspector, dbName)
	if err != nil {
		log.Fatalf("inspect schema: %v", err)
	}
	if len(schema.ForeignKeys) == 0 {
		fmt.Println("  (none)")
	}
	for _, fk := range schema.ForeignKeys {
		fmt.Printf("  %s.%s → %s.%s\n", fk.FromTable, fk.FromColumn, fk.ToTable, fk.ToColumn)
	}

	// --- Error handling demo ---
	fmt.Println("\n⚠️  Error handling (bad table):")
	_, err = db.Query(ctx, "SELECT * FROM non_existent_table_xyz")
	if err != nil {
		if e, ok := err.(*database.DBError); ok {
			fmt.Printf("  Kind=%v Message=%s\n", e.Kind, e.Message)
		}
	}

	fmt.Printf("\n✅ Done at %s\n", time.Now().Format(time.RFC3339))
}

func buildFlags(col database.ColumnInfo) string {
	var flags []string
	if col.IsPrimaryKey {
		flags = append(flags, "PK")
	}
	if col.IsUnique {
		flags = append(flags, "UNIQUE")
	}
	if !col.IsNullable {
		flags = append(flags, "NOT NULL")
	}
	if col.DefaultValue != nil {
		flags = append(flags, fmt.Sprintf("DEFAULT=%s", *col.DefaultValue))
	}
	return strings.Join(flags, " ")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
