package main

import (
	"github.com/p2pictomania/p2pictomania/bootstrap"
	"github.com/p2pictomania/p2pictomania/connections"
	"github.com/p2pictomania/p2pictomania/web"
	"log"
	"os"
	"strconv"
)

func main() {

	if len(os.Args) < 3 {
		log.Println("Usage: ./p2pictomania portno nodename")
		return
	}

	tempPort, err := strconv.Atoi(os.Args[1])

	if err != nil {
		log.Println("Usage: ./p2pictomania portno nodename")
		return
	}

	tempName := os.Args[2]

	//set global NodeNickName, NodeListenPort and EncryptionKey
	connections.SetListenPort(tempPort)
	connections.SetNickName(tempName)
	connections.SetEncryptionKey("djreglhvbcnfqstuwxymkpioaz456789")

	go web.StartServer()
	go bootstrap.StartServer()

	connections.InitSocketCache(&connections.Sc)
	go connections.ServerListener(connections.NodeListenPort)

	//block forever
	select {}
}
