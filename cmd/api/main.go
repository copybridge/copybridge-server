package main

import (
	"fmt"

	"github.com/copybridge/copybridge-server/internal/server"
)

func main() {

	server := server.NewServer()

	fmt.Printf("Starting server on %s...", server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("cannot start server: %s", err))
	}
}
