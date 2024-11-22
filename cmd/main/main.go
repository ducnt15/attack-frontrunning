package main

import (
	"attack-frontrunning/internal/service"
	"fmt"
)

func main() {
	err := service.ContractInteract()
	if err != nil {
		fmt.Println(err)
		return
	}
}
