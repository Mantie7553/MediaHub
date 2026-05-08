package main

import (
	"fmt"
	"log"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/database"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/server"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := database.New()
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}
	defer db.Close()

	fmt.Println("Database connected successfully")

	s := server.New(db)
	fmt.Println("MediaHub API starting on port 9090...")
	if err := s.Start(":9090"); err != nil {
		log.Fatal(err)
	}
}
