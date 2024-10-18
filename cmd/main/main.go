package main

import (
	"attack-frontrunning/internal/service"
	"fmt"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

func main() {
	err := service.ContractInteract()
	if err != nil {
		return
	}
}
