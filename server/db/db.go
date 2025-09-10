package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Pritam-deb/echo-sense/db/models"
	_ "github.com/lib/pq" // needed for sql.Open
)

var DB *gorm.DB

func CreateDBIfNotExists() {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	// password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	// Connect to default postgres database
	dsn := fmt.Sprintf(
		"host=%s user=%s dbname=postgres port=%s sslmode=disable",
		host, user, port,
	)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer sqlDB.Close()

	// Check if db exists
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	err = sqlDB.QueryRow(query, dbname).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check if database exists: %v", err)
	}

	if !exists {
		_, err = sqlDB.Exec("CREATE DATABASE " + dbname)
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		log.Printf("Database %s created successfully 🎉", dbname)
	}
}

func Connect() {
	CreateDBIfNotExists()

	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	// password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf(
		"host=%s user=%s dbname=%s port=%s sslmode=disable",
		host, user, dbname, port,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// Run migrations
	err = DB.AutoMigrate(&models.Song{}, &models.AudioFingerprint{})
	if err != nil {
		log.Fatal("Failed to run migrations: ", err)
	}

	log.Println("Database connection successful 🚀")
}
