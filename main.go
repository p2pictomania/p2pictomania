package main

import (
	"log"

	"github.com/p2pictomania/p2pictomania/bootstrap"
	"github.com/p2pictomania/p2pictomania/web"
)

func main() {
	go web.StartServer()
	bootstrap.StartServer()
	log.Println("Server started successfully")
}
