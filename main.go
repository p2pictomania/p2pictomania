package main

import (
	"bufio"
	"fmt"
	"github.com/p2pictomania/p2pictomania/bootstrap"
	"github.com/p2pictomania/p2pictomania/connections"
	"github.com/p2pictomania/p2pictomania/web"
	"log"
	"os"
	//"os/signal"
	"strconv"
	//"syscall"
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

	/*
		sigs := make(chan os.Signal, 1)
		//done := make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	*/

	//Fixed listening port for every node
	var tempPort int = 1111

	connections.SetIP(selfIP)
	connections.SetListenPort(tempPort)
	connections.SetNickName(tempName)
	connections.SetEncryptionKey("djreglhvbcnfqstuwxymkpioaz456789")
	connections.SetCurrentRoom(-1)

	log.Println("Node info: Name=" + connections.NodeNickName + " IP=" + connections.NodeIP + " ListenPort=" + strconv.Itoa(connections.NodeListenPort))

	go web.StartServer()
	go bootstrap.StartServer()

	connections.InitSocketCache(&connections.Sc)
	go connections.ServerListener(connections.NodeListenPort)

	reader := bufio.NewReader(os.Stdin)
	/*
		go func() {
			sig := <-sigs
			bootstrap.Wg.Add(1)
			fmt.Println()
			fmt.Println(sig)
			log.Println("Performing cleanup")
			bootstrap.DeleteSelfFromDNS()
			bootstrap.Wg.Wait()
			log.Println("Cleanup complete")
			os.Exit(0)
			//done <- true
		}()
	*/
	for {
		fmt.Println("Enter command:")
		cmd, _ := reader.ReadString('\n')

		switch cmd {

		case "join\n":
			var input int
			fmt.Print("Enter roomNumber to join:")
			_, err := fmt.Scanf("%d", &input)

			if err != nil {
				fmt.Println("Invalid roomNumber")
				continue
			}

			connections.JoinRoom(connections.NodeNickName, strconv.Itoa(input))

		case "send\n":
			fmt.Println("Enter data:")
			content, _ := reader.ReadString('\n')
			currentgroup := strconv.Itoa(connections.NodeRoomID)
			//msg := connections.Message{SenderIP: connections.NodeIP, SenderPort: connections.NodeListenPort, Kind: "Data", Originator: connections.NodeNickName, Data: content}
			msg := connections.Message{SenderIP: connections.NodeIP, SenderPort: connections.NodeListenPort, Groupname: currentgroup, Kind: "Data", Originator: connections.NodeNickName, Data: content}
			connections.Send(msg)

		case "quit\n":
			os.Exit(0)

		case "leave\n":
			var input int
			fmt.Print("Enter roomNumber to leave:")
			_, err := fmt.Scanf("%d", &input)

			if err != nil {
				fmt.Println("Invalid roomNumber")
				continue
			}
			connections.ExitRoom(input)

		default:
			fmt.Println("Unsupported command")
		}

	}

	//block forever
	select {}
}
