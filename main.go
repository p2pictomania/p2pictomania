package main

import (
	"bufio"
	"fmt"
	"github.com/p2pictomania/p2pictomania/bootstrap"
	"github.com/p2pictomania/p2pictomania/connections"
	"github.com/p2pictomania/p2pictomania/web"
	"log"
	"os"
	"strconv"
)

func main() {

	if len(os.Args) < 2 {
		log.Println("Usage: ./p2pictomania nodename")
		return
	}

	/*
		tempPort, err := strconv.Atoi(os.Args[1])

		if err != nil {
			log.Println("Usage: ./p2pictomania portno nodename")
			return
		}
	*/

	tempName := os.Args[1]

	//set global NodeNickName, NodeListenPort and EncryptionKey
	selfIP, err := bootstrap.GetPublicIP()

	if err != nil {
		log.Println("Error while fetching public IP of node")
		log.Println("Please restart application")
		os.Exit(0)
		//panic(err.Error())
	}

	//Fixed listening port for every node
	var tempPort int = 1111

	connections.SetIP(selfIP)
	connections.SetListenPort(tempPort)
	connections.SetNickName(tempName)
	connections.SetEncryptionKey("djreglhvbcnfqstuwxymkpioaz456789")

	log.Println("Node info: Name=" + connections.NodeNickName + " IP=" + connections.NodeIP + " ListenPort=" + strconv.Itoa(connections.NodeListenPort))

	go web.StartServer()
	go bootstrap.StartServer()

	connections.InitSocketCache(&connections.Sc)
	go connections.ServerListener(connections.NodeListenPort)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter command:")
		cmd, _ := reader.ReadString('\n')

		switch cmd {

		case "join\n":
			/*
				var input int
				fmt.Print("Enter roomNumber:")
				_, err := fmt.Scanf("%d", &input)
				if err != nil {
					fmt.Println("Invalid roomNumber")
					continue
				}
			*/
			connections.JoinRoom(connections.NodeNickName, "1")

		case "send\n":
			fmt.Println("Enter data:")
			content, _ := reader.ReadString('\n')
			msg := connections.Message{SenderIP: connections.NodeIP, SenderPort: connections.NodeListenPort, Kind: "Data", Originator: connections.NodeNickName, Data: content}
			connections.Send(msg)

		case "quit\n":
			os.Exit(0)

		case "leave\n":
			connections.ExitRoom("room1")

		default:
			fmt.Println("Unsupported command")
		}

	}

	//block forever
	select {}
}
