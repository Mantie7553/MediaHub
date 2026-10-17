package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/Mantie7553/MediaHub/backend/internal/cleanup"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/database"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/server"
	"github.com/joho/godotenv"
)

func main() {
	// optional, gitignored overrides for local testing (e.g. MEDIA_ROOT on local storage).
	// godotenv never overwrites a variable that is already set, so this must load before .env
	if err := godotenv.Load(".env.local"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatal("Error loading .env.local file: ", err)
	}
	// in Docker the config comes from the container environment, so .env is optional
	if err := godotenv.Load(); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Fatal("Error loading .env file: ", err)
		}
		fmt.Println("No .env file, using environment variables")
	}

	db, err := database.New()
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}
	defer db.Close()

	fmt.Println("Database connected successfully")

	if err := database.Migrate(db); err != nil {
		log.Fatal("Error running migrations: ", err)
	}
	if changed, err := database.SetInitialAdminPassword(db); err != nil {
		log.Fatal("Error setting admin password: ", err)
	} else if changed {
		fmt.Println("Default admin password replaced with ADMIN_PASSWORD")
	}

	s := server.New(db)
	cleanup.Start(os.Getenv("MEDIA_ROOT"))
	fmt.Println("MediaHub API starting on port 9090...")
	if err := s.Start(":9090"); err != nil {
		log.Fatal(err)
	}
}
