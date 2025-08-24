package database

import (
	"context"
	"fmt"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

var q *Queries

func Init() {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		config.Env.DBHost, config.Env.DBPort,
		config.Env.DBName, config.Env.DBUser,
		config.Env.DBPassword,
	)
	runMigrations(dsn)
	pool := getConnectionPool(dsn)

	// convert pool to sqlc queries
	q = New(pool)
}

func getConnectionPool(dsn string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		logger.L.Fatalf("failed to connect to database: %v", err)
	}
	return pool
}

func Q() *Queries {
	return q
}
