package database

import (
	"embed"

	"github.com/govdbot/govd/internal/logger"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type gooseLogger struct {
	log *zap.SugaredLogger
}

func (l gooseLogger) Fatalf(format string, v ...interface{}) {
	l.log.Fatalf(format, v...)
}

func (l gooseLogger) Printf(format string, v ...interface{}) {
	l.log.Infof(format, v...)
}

func runMigrations(dsn string) {
	goose.SetBaseFS(embedMigrations)
	goose.SetLogger(gooseLogger{log: logger.L})
	goose.SetDialect("postgres")

	db, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		logger.L.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	err = goose.Up(db, "migrations")
	if err != nil {
		logger.L.Fatalf("failed to run migrations: %v", err)
	}
}
