package main

import (
	"github.com/p2pictomania/p2pictomania/bootstrap"
	"github.com/p2pictomania/p2pictomania/game"
	"github.com/p2pictomania/p2pictomania/web"
)

func main() {

	go web.StartServer()
	go bootstrap.StartServer()
	go game.StartServer()

	//block forever
	select {}
}
