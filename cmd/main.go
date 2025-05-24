package main

import (
	"idm/inner/database"
	"log"
)

func main() {
	db, err := database.ConnectDb()
	if err != nil {
		log.Fatalf("Connection error: %v", err)
	}
	_ = db
}
