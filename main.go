package main

import (
	"bootstrap"
	"connections"
	"web"
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
    
    
	connections.NodeListenPort = tempPort
	connections.NodeNickName = tempName
	go web.StartServer()
	go bootstrap.StartServer()

	connections.InitSocketCache(&connections.Sc)
	go connections.ServerListener(connections.NodeListenPort)



	//block forever
	select {}
}
