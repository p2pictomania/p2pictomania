package connections

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	//"strings"
	//"os"
	"log"
	"reflect"
	"sync"
)

//TODO: cross compile all code
//TODO: implement heart beats (send-ack)
//Socket cache to hold connected sockets to other peers (Peers are identified by unique nickname)
type SocketCache struct {
	v   map[string]net.Conn
	mux sync.Mutex
}

//this struct might be shared with the game logic
type RoomMember struct {
	IP         string
	ListenPort int
	NickName   string
}

//global declarations
//socketCache
var Sc SocketCache

//should get this by a DNS query ideally
var BootstrapServerIP string = "127.0.0.1"
var BootstrapServerPort int = 5000

//these are for this peer (Read from config file or can be static)
var NodeIP string = "127.0.0.1"
var NodeListenPort int = 0
var NodeNickName string = "alice"

func InitSocketCache(sc *SocketCache) {
	sc.v = make(map[string]net.Conn)
}

//get a client socket from socket cache
func (c *SocketCache) Get(key string) net.Conn {

	c.mux.Lock()
	defer c.mux.Unlock()
	return c.v[key]
}

//insert a new client socket into the socket cache
func (c *SocketCache) Put(key string, clientsock net.Conn) {

	c.mux.Lock()
	c.v[key] = clientsock
	c.mux.Unlock()

}

func (c *SocketCache) Delete(sock net.Conn) {

	c.mux.Lock()
	for key, value := range c.v {
		if reflect.DeepEqual(value, sock) {
			delete(c.v, key)
			break
		}
	}
	c.mux.Unlock()
}

type Timestruct struct {
	Timestamp int
}

type Message struct {
	SenderIP   string
	SenderPort int
	DestIP     string
	DestPort   int
	Kind       string
	Groupname  string
	Timestamp  Timestruct
	Cast       string
	Originator string
	SeqNo      int
	Data       string
}

//TODO: ensure that this function returns back (should not block due to anything e.g. channels, sockets etc) since this is called as part of receive
func handleMessage(msg Message, clientsock net.Conn) {

	if msg.Kind == "Connect" {
		//fmt.Println("Inside handle connect message")

		sock := Sc.Get(msg.Originator)

		//already have a valid socket for the nickname
		if sock != nil {
			fmt.Println("Already have a valid socket for " + msg.Originator)
			return
		}

		//save socket
		Sc.Put(msg.Originator, clientsock)
		fmt.Println("Socket saved for " + msg.Originator)

		//send connect message back for the other node

		replymsg := &Message{SenderIP: NodeIP, SenderPort: NodeListenPort, DestIP: msg.SenderIP, DestPort: msg.SenderPort, Kind: "Connect", Originator: NodeNickName}
		text, err := json.Marshal(replymsg)

		if err != nil {
			fmt.Println(err)
			fmt.Println("Unable to send connect message back to " + msg.Originator)
			return
		}

		fmt.Printf("Connect Back Message=%+v\n", replymsg)
		clientsock.Write([]byte(string(text) + "\n"))

	}

	//handle other types of messages

}

/*continuously receives data from the specified socket
  and pushes to the relevant channel for the layer above connections.
  Itself handles only connection specific messages
*/
func receive(clientsock net.Conn) {

	fmt.Println("Starting receive goroutine")

	for {

		//listen for message to process ending in newline
		message, err := bufio.NewReader(clientsock).ReadString('\n')

		if err != nil {
			fmt.Println("Socket error.")
			clientsock.Close()
			Sc.Delete(clientsock)
			break
		}

		fmt.Print("Message received:", string(message))

		res := Message{}
		err = json.Unmarshal([]byte(string(message)), &res)

		if err != nil {
			fmt.Println(err)
		}

		//can print a struct with +v
		fmt.Printf("res=%+v\n", res)

		//handles connection related message. Pass on other messages to the layer above
		handleMessage(res, clientsock)

		/*
			newmessage := strings.ToUpper(message)

			//send new string back to client
			clientsock.Write([]byte(newmessage + string('\n')))
		*/
	}
}

func GetRoomslist(nickname string) string {

	//TODO: make HTTP request/or connect socket to bootstrap server and get the list of rooms and number of members in each

	return "RoomID: Room1 Members:1 RoomID: Room2 Members:2"

}

//contacts the bootstrap server to join a room. Get a list of (IP,Port) already in the room and connects to them
func JoinRoom(nickname string, roomName string) {

	//TODO: make HTTP request/or connect socket to bootstrap server, send self nickname and get a list of (IP,Ports,nicknames)
	var membersList [5]RoomMember
	rm1 := RoomMember{IP: "127.0.0.1", ListenPort: 1111, NickName: "bob"}
	rm2 := RoomMember{IP: "127.0.0.1", ListenPort: 2222, NickName: "alice"}
	rm3 := RoomMember{IP: "127.0.0.1", ListenPort: 3333, NickName: "daphnie"}
	membersList[0] = rm1
	membersList[1] = rm2
	membersList[2] = rm3

	fmt.Println("within joinRoom function")

	for i := 0; i < len(membersList); i++ {

		if membersList[i].ListenPort == 0 {
			fmt.Println("Skipping empty entry in the membersList")
			continue
		}

		//also check to skip the case of the peer connecting to itself (same nickname)
		if membersList[i].NickName == NodeNickName {
			fmt.Println("Skipping connection to itself in the group")
			continue
		}

		sock := Sc.Get(membersList[i].NickName)

		//already have a valid socket for the nickname
		if sock != nil {
			fmt.Println("Skipping node as a valid socket is found")
			continue
		}

		//establish a new connection to the room member if we have no valid socket for it
		fmt.Println("Connecting to node at " + membersList[i].IP + " " + strconv.Itoa(membersList[i].ListenPort))
		conn, err := net.Dial("tcp", membersList[i].IP+":"+strconv.Itoa(membersList[i].ListenPort))

		if err != nil {
			fmt.Println("Unable to connect to node at " + membersList[i].IP + " " + strconv.Itoa(membersList[i].ListenPort))
			continue
		}

		msg := &Message{SenderIP: NodeIP, SenderPort: NodeListenPort, DestIP: membersList[i].IP, DestPort: membersList[i].ListenPort, Kind: "Connect", Originator: NodeNickName}
		text, err := json.Marshal(msg)

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("connectMessage=%+v\n", msg)
		//save the socket and start a receiving goroutine for the new socket
		Sc.Put(membersList[i].NickName, conn)
		go receive(conn)
		//send connect message
		conn.Write([]byte(string(text) + "\n"))

	}
}

//finds the socket for the nickname, marshals the message into json and sends it out to the client
func send(msg Message, nickname string) {

	clientsock := Sc.Get(nickname)

	if clientsock != nil {

		text, err := json.Marshal(msg)

		if err != nil {
			fmt.Println(err)
			return
		}

		clientsock.Write([]byte(string(text) + "\n"))

	}

	fmt.Println("Error from send(): clientsock is nil")

}

//listens to incoming client connections
func ServerListener(listeningport int) {

	log.Println("Listening for connections at port " + strconv.Itoa(listeningport))

	//listen on all interfaces on port listeningport
	ln, _ := net.Listen("tcp", ":"+strconv.Itoa(listeningport))

	for {
		//accept client connection
		conn, _ := ln.Accept()

		//start receiver thread
		go receive(conn)
	}
}
