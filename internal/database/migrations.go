package database

import (
	"database/sql"
	"embed"

	"github.com/govdbot/govd/internal/logger"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

func openDB() (*sql.DB, error) {
	dsn := getDSN()
	goose.SetBaseFS(MigrationsFS)
	goose.SetLogger(gooseLogger{log: logger.L})
	goose.SetDialect("postgres")

	db, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func runMigrations() {
	db, err := openDB()
	if err != nil {
		logger.L.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	err = goose.Up(db, "migrations")
	if err != nil {
		logger.L.Fatalf("failed to run migrations: %v", err)
	}
}
