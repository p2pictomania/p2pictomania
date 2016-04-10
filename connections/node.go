package connections

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	zmq "github.com/pebbe/zmq4"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

//TODO: cross compile all code
//TODO: implement heart beats (send-ack)

//SocketCache to hold connected sockets to other peers (Peers are identified by unique nickname)
type SocketCache struct {
	v   map[string]zmq.Socket
	mux sync.Mutex
}

//RoomMember might be shared with the game logic
type RoomMember struct {
	IP         string
	ListenPort int
	NickName   string
}

//TODO: should ideally refer this struct from the boostrap module
type addPlayerToRoomJSON struct {
	RoomID         int    `json:"roomID"`
	PlayerNickName string `json:"nickName"`
	PlayerIP       string `json:"playerIP"`
}

//TODO: should ideally refer this struct from the boostrap module
type deletePlayerFromRoomJSON struct {
	RoomID         int    `json:"roomID"`
	PlayerNickName string `json:"nickName"`
}

type queryResults struct {
	Results []struct {
		Columns []string        `json:"columns"`
		Types   []string        `json:"types"`
		Values  [][]interface{} `json:"values"`
		Time    float64         `json:"time"`
	} `json:"results"`
}

type status struct {
	Status int `json:"status"`
}

//global declarations
//socketCache
var Sc SocketCache

//should get this by a DNS query ideally
//var BootstrapServerIP string = "127.0.0.1"
//var BootstrapServerPort int = 5000

//these are for this peer (Read from config file or can be static)
var NodeIP string = "127.0.0.1"
var NodeListenPort int = 1111
var NodeNickName string = "randomname"
var PubSocket *zmq.Socket
var key = []byte("0")
var NodeRoomID int = -1

func IsClosedSocket(sock zmq.Socket) bool {
	return strings.Contains(sock.String(), "CLOSED")
}

func InitSocketCache(sc *SocketCache) {
	sc.v = make(map[string]zmq.Socket)
}

//get a client socket from socket cache
func (c *SocketCache) Get(key string) zmq.Socket {

	c.mux.Lock()
	defer c.mux.Unlock()
	return c.v[key]
}

//insert a new client socket into the socket cache
func (c *SocketCache) Put(key string, clientsock zmq.Socket) {

	c.mux.Lock()
	c.v[key] = clientsock
	c.mux.Unlock()

}

func (c *SocketCache) Delete(sock net.Conn) {

	c.mux.Lock()
	for key, value := range c.v {
		//TODO: check whether DeepEqual works for zmq sockets or change this part to work with zmq
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

//TODO: push data to a shared channel
func receiveFromPublisher(subSocket *zmq.Socket) {

	log.Println("Started goreceiveFromPublisher for zmq subscriber socket")

	for {
		data, _ := subSocket.RecvMessage(0)

		//TODO:if socket is valid i.e not deleted, receive the message, else discard it
		//closing the socket causes panic so this is avoided by checking the validity of the socket in the cache
		res := Message{}

		if len(data) == 0 {
			log.Println("Empty message received")
			continue
		}

		err := json.Unmarshal([]byte(string(data[0])), &res)

		if err != nil {
			log.Println("Error while unmarshalling in receive" + err.Error())
			continue
		}

		/*
			if IsClosedSocket(Sc.v[res.Originator]) == false {

				log.Printf("res=%+v\n", res)
				//log.Print("Message from publisher:")
				//log.Println(data)
			}
		*/

		currentroom := strconv.Itoa(NodeRoomID)

		if res.Groupname == currentroom {
			log.Printf("res=%+v\n", res)
		}

		//can print a struct with +v

		//Uncomment the lines below to enable decryption
		/*
			decryptedData, err := Decrypt(key, []byte(data[0]))
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(string(decryptedData))
		*/
	}

}

func deleteSelfFromRoom(selfHostName string, roomID int) int {

	listOfBootstrapNodes, _ := net.LookupHost("autogra.de")
	bootstrapip, err := GetLeaderIP(listOfBootstrapNodes)
	if err != nil {
		return http.StatusInternalServerError
	}

	var url = "http://" + bootstrapip + ":5000/player/leave"
	msg := deletePlayerFromRoomJSON{roomID, selfHostName}
	jsonStr, err := json.Marshal(msg)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Close = true
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	//log.Println(resp)
	if err != nil {
		log.Println("Error while sending POST request to leave room")
		log.Println(err)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println("Error from ReadAll in deleteSelfFromRoom:" + err.Error())
		panic(err.Error())
	}

	log.Println("Response Body for bootstrap request=" + string(body))

	resultjson := status{}
	json.Unmarshal(body, &resultjson)

	log.Println("Status response from server in deleteSelfFromRoom:" + strconv.Itoa(resultjson.Status))

	return resultjson.Status

}

//registers an (IP, Hostname and roomID) with the bootstrap servers
func insertSelfIntoRoom(selfIP string, selfHostName string, roomID int) int {

	listOfBootstrapNodes, _ := net.LookupHost("autogra.de")
	bootstrapip, err := GetLeaderIP(listOfBootstrapNodes)
	if err != nil {
		return http.StatusInternalServerError
	}

	var url = "http://" + bootstrapip + ":5000/player/join"
	msg := addPlayerToRoomJSON{roomID, selfHostName, selfIP}
	jsonStr, err := json.Marshal(msg)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Close = true
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	//log.Println(resp)
	if err != nil {
		log.Println("Error while sending POST request to join room")
		log.Println(err)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println("Error from ReadAll in InsertSelfFromRoom:" + err.Error())
		panic(err.Error())
	}

	log.Println("Response Body for bootstrap request=" + string(body))

	resultjson := status{}
	json.Unmarshal(body, &resultjson)

	log.Println("Status response from server in InsertSelfFromRoom:" + strconv.Itoa(resultjson.Status))
	return resultjson.Status

}

//Returns membersList and number of members in the room
func queryRoom(roomID int) ([5]RoomMember, int, error) {

	var membersList [5]RoomMember

	listOfBootstrapNodes, _ := net.LookupHost("autogra.de")
	bootstrapip, err := GetLeaderIP(listOfBootstrapNodes)
	if err != nil {
		return membersList, 0, errors.New("No Leader found to execute query")
	}

	url := "http://" + bootstrapip + ":5000/peers/" + strconv.Itoa(roomID)

	log.Println("QueryRoom URL=" + url)

	res, err := http.Get(url)

	if err != nil {
		log.Println("Error in queryRoom get request:" + err.Error())

		if e, okay := err.(net.Error); okay && e.Timeout() {
			log.Println("Timeout error while querying the room")
			return membersList, 0, err
		}
		panic(err.Error())
		//return
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		log.Println("error in readall")
		panic(err.Error())
	}

	stripDoubleQuotesBody := string(body)
	resultjson := queryResults{}
	json.Unmarshal([]byte(stripDoubleQuotesBody), &resultjson)

	log.Printf("%+v", resultjson)
	log.Println()

	if !strings.Contains(string(body), "values") {
		log.Println("This is the first node in the room")
		//skip establishing connections, membersList has nothing (all 0s) right now
		return membersList, 0, nil
	}

	//log.Println(len(resultjson.Results[0].Values))

	for i := 0; i < len(resultjson.Results[0].Values); i++ {
		rm1 := RoomMember{IP: resultjson.Results[0].Values[i][2].(string), ListenPort: 1111, NickName: resultjson.Results[0].Values[i][1].(string)}
		membersList[i] = rm1
		log.Printf("%+v", rm1)
		log.Println()
	}

	log.Print("membersList from queryRoom=")
	log.Println(membersList)
	return membersList, len(resultjson.Results[0].Values), nil
}

func ExitRoom(roomID int) {

	//TODO: make HTTP request/or connect socket to bootstrap server and get a list of nicknames for a room

	membersList, numMembers, errRoom := queryRoom(roomID)

	if errRoom != nil {
		log.Print("Error while querying the room in ExitRoom. Room not exited:" + errRoom.Error())
		return
	}

	//nicknameList := [5]string{"alice", "bob", "charlie", "daphnie", ""}
	_ = membersList
	_ = numMembers
	/*
		for i := 0; i < numMembers; i++ {
			if IsClosedSocket(Sc.v[membersList[i].NickName]) == false {
				//TODO: Close() causes panic in the receiveFromPublisher
				//nickSock := Sc.Get(nicknameList[i])
				//nickSock.Close()
				delete(Sc.v, membersList[i].NickName)
				log.Println("ExitRoom: subscriber socket closed for " + membersList[i].NickName)
			}
		}
	*/

	//make POST request to bootstrap server to remove self from the room
	//TODO: check return code for error. set roomID to -1 only when the deleteSelfFromRoom succeeds
	deleteSelfFromRoom(NodeNickName, roomID)

	SetCurrentRoom(-1)

}

func GetRoomslist(nickname string) string {

	//TODO: make HTTP request/or connect socket to bootstrap server and get the list of rooms and number of members in each

	return "RoomID: Room1 Members:1 RoomID: Room2 Members:2"

}

//contacts the bootstrap server to join a room. Get a list of (IP,Port) already in the room and connects to them
func JoinRoom(nickname string, roomName string) {

	//TODO: make HTTP request/or connect socket to bootstrap server and get a list of (IP,Ports,nicknames)

	/*
		var membersList [5]RoomMember

		rm1 := RoomMember{IP: "127.0.0.1", ListenPort: 1111, NickName: "bob"}
		rm2 := RoomMember{IP: "127.0.0.1", ListenPort: 2222, NickName: "alice"}
		rm3 := RoomMember{IP: "127.0.0.1", ListenPort: 3333, NickName: "daphnie"}
		membersList[0] = rm1
		membersList[1] = rm2
		membersList[2] = rm3
	*/

	roomID, err := strconv.Atoi(roomName)

	if err != nil {
		log.Println("Invalid room name:" + roomName)
		return
	}

	membersList, numMembers, errRoom := queryRoom(roomID)

	if errRoom != nil {
		log.Print("Error while querying the room. Room not joined:")
		log.Println(errRoom)
		return
	}

	insertSelfIntoRoom(NodeIP, NodeNickName, roomID)

	log.Println("within joinRoom function")

	for i := 0; i < numMembers; i++ {

		log.Println("Nick=" + membersList[i].NickName)

		if membersList[i].ListenPort == 0 {
			log.Println("Skipping empty entry in the membersList")
			continue
		}

		//also check to skip the case of the peer connecting to itself (same nickname)
		if membersList[i].NickName == NodeNickName {
			log.Println("Skipping connection to itself in the group")
			continue
		}

		sock := Sc.Get(membersList[i].NickName)
		//already have a valid socket for the nickname
		if IsClosedSocket(sock) == false {
			log.Println("Skipping node as a valid socket is found")
			continue
		}

		//establish a new connection to the room member if we have no valid socket for it
		log.Println("Connecting to node at " + membersList[i].IP + " " + strconv.Itoa(membersList[i].ListenPort))

		clientSubSock, _ := zmq.NewSocket(zmq.SUB)

		clientSubSock.SetSubscribe("")
		clientSubSock.Connect("tcp://" + membersList[i].IP + ":" + strconv.Itoa(membersList[i].ListenPort))

		//Send CONNECT message to the other node for it to subscribe to this node
		conn, err := net.Dial("tcp", membersList[i].IP+":"+strconv.Itoa(membersList[i].ListenPort+1))

		if err != nil {
			log.Println("Unable to connect to node at " + membersList[i].IP + " " + strconv.Itoa(membersList[i].ListenPort+1))
			continue
		}

		msg := &Message{SenderIP: NodeIP, SenderPort: NodeListenPort, DestIP: membersList[i].IP, DestPort: membersList[i].ListenPort + 1, Kind: "Connect", Originator: NodeNickName}
		text, err := json.Marshal(msg)

		if err != nil {
			log.Println("Error while marshalling connect message" + err.Error())
			continue
		}

		log.Printf("connectMessage=%+v\n", msg)
		//send connect message
		conn.Write([]byte(string(text) + "\n"))

		//save the socket and start a receiving goroutine for the new socket
		Sc.Put(membersList[i].NickName, *clientSubSock)

		go receiveFromPublisher(clientSubSock)

	}

	SetCurrentRoom(roomID)
}

//TODO: ensure that this function returns back (should not block due to anything e.g. channels, sockets etc) since this is called as part of receive
func handleMessage(msg Message, clientsock net.Conn) {

	//TODO: find a better string comparison function (EqualsIgnoreCase)
	if msg.Kind == "Connect" {
		//fmt.Println("Inside handle connect message")

		sock := Sc.Get(msg.Originator)

		//already have a valid socket for the nickname
		if IsClosedSocket(sock) == false {
			log.Println("Already have a valid socket for " + msg.Originator)
			return
		}

		subSocket, _ := zmq.NewSocket(zmq.SUB)
		subSocket.SetSubscribe("")
		subSocket.Connect("tcp://" + msg.SenderIP + ":" + strconv.Itoa(msg.SenderPort))

		log.Println("Connect to publisher at " + msg.SenderIP + ":" + strconv.Itoa(msg.SenderPort))
		//save socket
		Sc.Put(msg.Originator, *subSocket)

		receiveFromPublisher(subSocket)

	}

	//handle other types of messages if required

}

/*continuously receives data from the specified socket
  and pushes to the relevant channel for the layer above connections.
  Itself handles only connection specific messages
*/
func receive(clientsock net.Conn) {

	log.Println("Starting receive goroutine")

	for {

		//listen for message to process ending in newline
		message, err := bufio.NewReader(clientsock).ReadString('\n')

		if err != nil {
			log.Println("Socket error in receive.")
			clientsock.Close()
			break
		}

		log.Print("Message received:", string(message))

		res := Message{}
		err = json.Unmarshal([]byte(string(message)), &res)

		if err != nil {
			log.Println("Error while unmarshalling in receive: " + err.Error())
		}

		//can print a struct with +v
		log.Printf("res=%+v\n", res)

		//handles connection related message. Pass on other messages to the layer above
		handleMessage(res, clientsock)

	}
}

//finds the socket for the nickname, marshals the message into json and sends it out to the client
func Send(msg Message) {

	log.Println("Within send")

	text, err := json.Marshal(msg)

	if err != nil {
		log.Println("Error while marshalling in send: " + err.Error())
		return
	}

	//Uncomment the below multiline comment to enable encryption
	/*
		ciphertext, err := Encrypt(key, text)

		if err != nil {
			fmt.Print("Unable to encrypt text: ")
			fmt.Println(err)
			return
		}

		PubSocket.SendMessage(string(ciphertext))
	*/

	PubSocket.SendMessage(string(text))

}

//ServerListener listens to incoming client connections and binds a port for the publisher
func ServerListener(listeningport int) {

	log.Println("Bind for publishing data at port " + strconv.Itoa(listeningport))
	log.Println("Listening for connections at port " + strconv.Itoa(listeningport+1))

	temppubsocket, _ := zmq.NewSocket(zmq.PUB)
	PubSocket = temppubsocket

	PubSocket.Bind("tcp://*:" + strconv.Itoa(listeningport))

	//listen on all interfaces on port listeningport
	ln, _ := net.Listen("tcp", ":"+strconv.Itoa(listeningport+1))

	for {
		//accept client connection
		conn, _ := ln.Accept()

		//start receiver thread
		go receive(conn)
	}
}

func SetNickName(nickName string) {
	NodeNickName = nickName
}

func SetListenPort(listenport int) {
	NodeListenPort = listenport
}

func SetIP(nodePublicIP string) {
	NodeIP = nodePublicIP
}

func SetEncryptionKey(enckey string) {
	key = []byte(enckey)
}

func SetCurrentRoom(RoomID int) {
	NodeRoomID = RoomID
}
