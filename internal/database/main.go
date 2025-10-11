package database

import (
	"context"
	"fmt"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool
var queries *Queries

func Init() {
	runMigrations()

	pool = getConnectionPool()

	// convert pool to sqlc queries
	queries = New(pool)
}

func getConnectionPool() *pgxpool.Pool {
	ctx := context.Background()
	dsn := getDSN()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.L.Fatalf("failed to connect to database: %v", err)
	}
	return pool
}

func getDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		config.Env.DBHost, config.Env.DBPort,
		config.Env.DBName, config.Env.DBUser,
		config.Env.DBPassword,
	)
}

func Q() *Queries {
	return queries
}

func Conn() *pgxpool.Pool {
	return pool
}
