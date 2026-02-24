// Package main demonstrates how to use DatRi's database layer with PostgreSQL.
//
// This example shows:
//   - Connecting to Postgres with a connection pool
//   - Schema introspection (listing tables and their columns)
//   - Building safe parameterized SELECT queries with the query builder
//   - Paginating results with LIMIT / OFFSET
//   - Scanning rows into Go maps
//   - Unified error handling with database.Is* predicates
//
// Run with:
//
//	DSN="postgres://user:pass@localhost:5432/mydb" go run examples/basic/postgres/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/koustreak/DatRi/internal/database"
	"github.com/koustreak/DatRi/internal/database/postgres"
)

func main() {
	// -----------------------------------------------------------------------
	// 1. Configuration
	// -----------------------------------------------------------------------
	dsn := os.Getenv("DSN")
	if dsn == "" {
		// Fallback for local development
		dsn = "postgres://postgres:postgres@localhost:5432/datri_example?sslmode=disable"
	}

	cfg := database.DefaultConfig(dsn)

	// Override pool settings if you need tighter control.
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.ConnectTimeout = 5 * time.Second
	cfg.QueryTimeout = 10 * time.Second

	// -----------------------------------------------------------------------
	// 2. Connect
	// -----------------------------------------------------------------------
	ctx := context.Background()

	db, err := postgres.New(ctx, cfg)
	if err != nil {
		if database.IsConnectionFailed(err) {
			log.Fatalf("could not reach postgres: %v", err)
		}
		log.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Connected to PostgreSQL")

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
			fmt.Printf("    %-20s %s%s%s\n", col.Name, col.DataType, nullable, pk)
		}
		if len(table.ForeignKeys) > 0 {
			for _, fk := range table.ForeignKeys {
				fmt.Printf("    FK: %s → %s.%s\n", fk.Column, fk.RefTable, fk.RefColumn)
			}
		}
		fmt.Println()
	}

	// -----------------------------------------------------------------------
	// 5. Check that a specific table exists before querying it
	// -----------------------------------------------------------------------
	targetTable := "users"
	exists, err := db.TableExists(ctx, targetTable)
	if err != nil {
		log.Fatalf("table check failed: %v", err)
	}
	if !exists {
		fmt.Printf("⚠️  Table %q does not exist — skipping query demo\n", targetTable)
		fmt.Println("\nCreate the table with:")
		fmt.Println("  CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL, email TEXT UNIQUE NOT NULL, active BOOLEAN DEFAULT true, created_at TIMESTAMPTZ DEFAULT now());")
		return
	}

	fmt.Printf("✅ Table %q found\n", targetTable)

	// -----------------------------------------------------------------------
	// 6. Query builder — SELECT with WHERE, ORDER BY, LIMIT, OFFSET
	// -----------------------------------------------------------------------
	sql, args, err := database.Select(targetTable, database.DialectPostgres).
		Columns("id", "name", "email", "created_at").
		Where("active", "=", true).
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
		if database.IsTimeout(err) {
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
	singleSQL, singleArgs, err := database.Select(targetTable, database.DialectPostgres).
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

	// For QueryRow we need to know the columns in advance (from schema cache).
	result, err := database.ScanRow(row, []string{"id", "name", "email"})
	if err != nil {
		if database.IsNotFound(err) {
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
