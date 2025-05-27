package database

import (
	"github.com/govdbot/govd/config"
	"github.com/govdbot/govd/models"

	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Start() {
	DB = connect()
	sqlDB, err := DB.DB()
	if err != nil {
		zap.S().Fatalf("failed to get database connection: %v", err)
	}
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	err = sqlDB.Ping()
	if err != nil {
		zap.S().Fatalf("failed to ping database: %v", err)
	}
	err = migrateDatabase()
	if err != nil {
		zap.S().Fatalf("failed to migrate database: %v", err)
	}
}

func connect() *gorm.DB {
	connectionString := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True",
		config.Env.DBUser, config.Env.DBPassword,
		config.Env.DBHost, config.Env.DBPort,
		config.Env.DBName,
	)
	zap.S().Debug("connecting to database")

	var conn *gorm.DB
	var err error

	maxRetries := 10
	retryCount := 0

	for retryCount < maxRetries {
		conn, err = gorm.Open(mysql.Open(connectionString), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
			NowFunc: func() time.Time {
				utc, _ := time.LoadLocation("Europe/Rome")
				return time.Now().In(utc)
			},
		})
		if err == nil {
			break
		}
		retryCount++
		zap.S().Warnf(
			"failed to connect to database (%d/%d)",
			retryCount, maxRetries,
		)
		if retryCount < maxRetries {
			time.Sleep(2 * time.Second)
		}
	}
	if err != nil {
		zap.S().Fatalf("failed to connect to database: %v", err)
	}
	return conn
}

func migrateDatabase() error {
	zap.S().Debug("migrating database")
	err := DB.AutoMigrate(
		&models.Media{},
		&models.MediaFormat{},
		&models.GroupSettings{},
		&models.User{},
	)
	if err != nil {
		return err
	}
	return nil
}
