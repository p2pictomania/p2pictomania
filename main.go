package main

import (
	"log"

	"github.com/p2pictomania/p2pictomania/supernode"
	"github.com/p2pictomania/p2pictomania/web"
)

func main() {
	go web.StartServer()
	supernode.StartServer()
	log.Println("Server started successfully")
}
