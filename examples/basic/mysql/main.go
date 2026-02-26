// Package main demonstrates how to use DatRi's database layer with MySQL.
//
// This example shows:
//   - Connecting to MySQL with a connection pool
//   - Schema introspection (listing tables and their columns)
//   - Building safe parameterized SELECT queries with the query builder
//   - Paginating results with LIMIT / OFFSET
//   - Scanning rows into Go maps
//   - Unified error handling with database.Is* predicates
//
// Run with:
//
//	DSN="user:pass@tcp(localhost:3306)/mydb?parseTime=true" go run examples/basic/mysql/main.go
//
// Note: parseTime=true in the DSN is required for proper time.Time scanning.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/koustreak/DatRi/internal/database"
	"github.com/koustreak/DatRi/internal/database/mysql"
	"github.com/koustreak/DatRi/internal/errs"
)

func main() {
	// -----------------------------------------------------------------------
	// 1. Configuration
	// -----------------------------------------------------------------------
	dsn := os.Getenv("DSN")
	if dsn == "" {
		// Fallback for local development.
		// parseTime=true → MySQL driver scans DATETIME/TIMESTAMP as time.Time
		// charset=utf8mb4 → full Unicode support (emoji, etc.)
		dsn = "root:root@tcp(localhost:3306)/datri_example?parseTime=true&charset=utf8mb4"
	}

	cfg := database.DefaultConfig(dsn)
	cfg.Driver = database.DriverMySQL

	// Override pool settings if needed.
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.ConnectTimeout = 5 * time.Second
	cfg.QueryTimeout = 10 * time.Second

	// -----------------------------------------------------------------------
	// 2. Connect
	// -----------------------------------------------------------------------
	ctx := context.Background()

	db, err := mysql.New(ctx, cfg)
	if err != nil {
		if errs.IsConnectionFailed(err) {
			log.Fatalf("could not reach mysql: %v", err)
		}
		log.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Connected to MySQL")

	// -----------------------------------------------------------------------
	// 3. Ping
	// -----------------------------------------------------------------------
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := db.Ping(pingCtx); err != nil {
		log.Fatalf("ping failed: %v", err)
	}
	fmt.Println("✅ Ping OK")

	// -----------------------------------------------------------------------
	// 4. Schema introspection
	//    In production, call this once at startup and cache the result.
	// -----------------------------------------------------------------------
	schema, err := db.InspectSchema(ctx)
	if err != nil {
		log.Fatalf("schema inspection failed: %v", err)
	}

	fmt.Printf("\n📋 Found %d table(s):\n", len(schema.Tables))
	for _, table := range schema.Tables {
		fmt.Printf("  • %s\n", table.Name)
		fmt.Printf("    PK: %v\n", table.PrimaryKey)
		for _, col := range table.Columns {
			nullable := ""
			if col.Nullable {
				nullable = " (nullable)"
			}
			pk := ""
			if col.IsPrimary {
				pk = " [PK]"
			}
			unique := ""
			if col.IsUnique {
				unique = " [UNI]"
			}
			fmt.Printf("    %-20s %s%s%s%s\n", col.Name, col.DataType, nullable, pk, unique)
		}
		if len(table.ForeignKeys) > 0 {
			for _, fk := range table.ForeignKeys {
				fmt.Printf("    FK: %s → %s.%s\n", fk.Column, fk.RefTable, fk.RefColumn)
			}
		}
		fmt.Println()
	}

	// -----------------------------------------------------------------------
	// 5. Check that the target table exists before querying
	// -----------------------------------------------------------------------
	targetTable := "users"
	exists, err := db.TableExists(ctx, targetTable)
	if err != nil {
		log.Fatalf("table check failed: %v", err)
	}
	if !exists {
		fmt.Printf("⚠️  Table %q does not exist — skipping query demo\n", targetTable)
		fmt.Println("\nCreate the table with:")
		fmt.Println("  CREATE TABLE users (")
		fmt.Println("    id         INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,")
		fmt.Println("    name       VARCHAR(255) NOT NULL,")
		fmt.Println("    email      VARCHAR(255) NOT NULL UNIQUE,")
		fmt.Println("    active     TINYINT(1) DEFAULT 1,")
		fmt.Println("    created_at DATETIME DEFAULT CURRENT_TIMESTAMP")
		fmt.Println("  );")
		return
	}

	fmt.Printf("✅ Table %q found\n", targetTable)

	// -----------------------------------------------------------------------
	// 6. Query builder — SELECT with WHERE, ORDER BY, LIMIT, OFFSET
	//    Note: DialectMySQL — placeholders are ? instead of $1, $2
	// -----------------------------------------------------------------------
	sql, args, err := database.Select(targetTable, database.DialectMySQL).
		Columns("id", "name", "email", "created_at").
		Where("active", "=", 1). // MySQL TINYINT(1), not bool
		OrderBy("created_at", database.Desc).
		Limit(10).
		Offset(0).
		Build()
	if err != nil {
		log.Fatalf("query build failed: %v", err)
	}

	fmt.Printf("\n🔍 Built query:\n  %s\n  args: %v\n\n", sql, args)

	// -----------------------------------------------------------------------
	// 7. Execute and scan rows
	// -----------------------------------------------------------------------
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		if errs.IsTimeout(err) {
			log.Fatalf("query timed out: %v", err)
		}
		log.Fatalf("query failed: %v", err)
	}

	results, err := database.ScanRows(rows)
	if err != nil {
		log.Fatalf("scan failed: %v", err)
	}

	fmt.Printf("📦 %d row(s) returned:\n", len(results))
	printJSON(results)

	// -----------------------------------------------------------------------
	// 8. QueryRow — fetch a single row by primary key
	// -----------------------------------------------------------------------
	singleSQL, singleArgs, err := database.Select(targetTable, database.DialectMySQL).
		Columns("id", "name", "email").
		Where("id", "=", 1).
		Build()
	if err != nil {
		log.Fatalf("query build failed: %v", err)
	}

	row, err := db.QueryRow(ctx, singleSQL, singleArgs...)
	if err != nil {
		log.Fatalf("query row failed: %v", err)
	}

	// For QueryRow we need to know the columns in advance (from the schema cache).
	result, err := database.ScanRow(row, []string{"id", "name", "email"})
	if err != nil {
		if errs.IsNotFound(err) {
			fmt.Println("⚠️  Row with id=1 not found")
		} else {
			log.Fatalf("scan row failed: %v", err)
		}
	} else {
		fmt.Println("\n👤 Single row (id=1):")
		printJSON(result)
	}
}

// printJSON is a helper that pretty-prints any value as JSON.
func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "  ", "  ")
	fmt.Printf("  %s\n", b)
}
