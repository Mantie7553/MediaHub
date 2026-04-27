package main

import (
	"fmt"
	"log"

	"github.com/Mantie7553/MediaHub/backend/internal/server"
)

func main() {
	s := server.New()
	fmt.Println("MediaHub API starting on port 8080")
	if err := s.Start(":9090"); err != nil {
		log.Fatal(err)
	}
}
