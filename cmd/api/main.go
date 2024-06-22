package main

import (
	"fmt"

	"github.com/copybridge/copybridge-server/internal/server"
)

func main() {

	server := server.NewServer()

	fmt.Println("Starting server...")
	err := server.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("cannot start server: %s", err))
	}
}
