package connections

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"io"
	"log"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

//TODO: cross compile all code
//TODO: implement heart beats (send-ack)
//Socket cache to hold connected sockets to other peers (Peers are identified by unique nickname)
type SocketCache struct {
	v   map[string]zmq.Socket
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
var NodeNickName string = "randomname"
var PubSocket *zmq.Socket
var key = []byte("0")

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

func Encrypt(key, text []byte) ([]byte, error) {

	//returns a new cipher for the key size
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//encode plain text to base64
	b := base64.StdEncoding.EncodeToString(text)

	ciphertext := make([]byte, aes.BlockSize+len(b))

	//initialization vector = first BlockSize bytes of ciphertext
	//IV's length = Block size
	iv := ciphertext[:aes.BlockSize]

	//iv = random bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	//stream for cipher feedback mode
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))

	return ciphertext, nil
}

func Decrypt(key, text []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)

	data, err := base64.StdEncoding.DecodeString(string(text))

	if err != nil {
		return nil, err
	}

	return data, nil
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

	fmt.Println("Started goreceiveFromPublisher for zmq subscriber socket")

	for {
		data, _ := subSocket.RecvMessage(0)
		fmt.Print("Message from publisher:")

		//Uncomment the lines below to enable decryption
		/*
			decryptedData, err := Decrypt(key, []byte(data[0]))
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(string(decryptedData))
		*/
		fmt.Println(data)
	}

}

func ExitRoom(roomName string) {

	//TODO: make HTTP request/or connect socket to bootstrap server and get a list of nicknames for a room

	nicknameList := [5]string{"alice", "bob", "charlie", "daphnie", ""}

	for i := 0; i < len(nicknameList); i++ {
		if IsClosedSocket(Sc.v[nicknameList[i]]) == false {
			//TODO: Close() causes panic in the receiveFromPublisher
			//nickSock := Sc.Get(nicknameList[i])
			//nickSock.Close()
			delete(Sc.v, nicknameList[i])
			fmt.Println("ExitRoom: subscriber socket closed for " + nicknameList[i])
		}
	}
}

func GetRoomslist(nickname string) string {

	//TODO: make HTTP request/or connect socket to bootstrap server and get the list of rooms and number of members in each

	return "RoomID: Room1 Members:1 RoomID: Room2 Members:2"

}

//contacts the bootstrap server to join a room. Get a list of (IP,Port) already in the room and connects to them
func JoinRoom(nickname string, roomName string) {

	//TODO: make HTTP request/or connect socket to bootstrap server and get a list of (IP,Ports,nicknames)
	var membersList [5]RoomMember
	rm1 := RoomMember{IP: "127.0.0.1", ListenPort: 1111, NickName: "bob"}
	rm2 := RoomMember{IP: "127.0.0.1", ListenPort: 2222, NickName: "alice"}
	rm3 := RoomMember{IP: "127.0.0.1", ListenPort: 3333, NickName: "daphnie"}
	membersList[0] = rm1
	membersList[1] = rm2
	membersList[2] = rm3

	fmt.Println("within joinRoom function")

	for i := 0; i < len(membersList); i++ {

		fmt.Println("Nick=" + membersList[i].NickName)

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
		if IsClosedSocket(sock) == false {
			fmt.Println("Skipping node as a valid socket is found")
			continue
		}

		//establish a new connection to the room member if we have no valid socket for it
		fmt.Println("Connecting to node at " + membersList[i].IP + " " + strconv.Itoa(membersList[i].ListenPort))

		clientSubSock, _ := zmq.NewSocket(zmq.SUB)

		clientSubSock.SetSubscribe("")
		clientSubSock.Connect("tcp://" + membersList[i].IP + ":" + strconv.Itoa(membersList[i].ListenPort))

		//Send CONNECT message to the other node for it to subscribe to this node
		conn, err := net.Dial("tcp", membersList[i].IP+":"+strconv.Itoa(membersList[i].ListenPort+1))

		if err != nil {
			fmt.Println("Unable to connect to node at " + membersList[i].IP + " " + strconv.Itoa(membersList[i].ListenPort+1))
			continue
		}

		msg := &Message{SenderIP: NodeIP, SenderPort: NodeListenPort, DestIP: membersList[i].IP, DestPort: membersList[i].ListenPort + 1, Kind: "Connect", Originator: NodeNickName}
		text, err := json.Marshal(msg)

		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf("connectMessage=%+v\n", msg)
		//send connect message
		conn.Write([]byte(string(text) + "\n"))

		//save the socket and start a receiving goroutine for the new socket
		Sc.Put(membersList[i].NickName, *clientSubSock)

		go receiveFromPublisher(clientSubSock)

	}
}

//TODO: ensure that this function returns back (should not block due to anything e.g. channels, sockets etc) since this is called as part of receive
func handleMessage(msg Message, clientsock net.Conn) {

	//TODO: find a better string comparison function (EqualsIgnoreCase)
	if msg.Kind == "Connect" {
		//fmt.Println("Inside handle connect message")

		sock := Sc.Get(msg.Originator)

		//already have a valid socket for the nickname
		if IsClosedSocket(sock) == false {
			fmt.Println("Already have a valid socket for " + msg.Originator)
			return
		}

		subSocket, _ := zmq.NewSocket(zmq.SUB)
		subSocket.SetSubscribe("")
		subSocket.Connect("tcp://" + msg.SenderIP + ":" + strconv.Itoa(msg.SenderPort))

		fmt.Println("Connect to publisher at " + msg.SenderIP + ":" + strconv.Itoa(msg.SenderPort))
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

	fmt.Println("Starting receive goroutine")

	for {

		//listen for message to process ending in newline
		message, err := bufio.NewReader(clientsock).ReadString('\n')

		if err != nil {
			fmt.Println("Socket error.")
			clientsock.Close()
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

	}
}

//finds the socket for the nickname, marshals the message into json and sends it out to the client
func send(msg Message) {

	fmt.Println("Within send")

	text, err := json.Marshal(msg)

	if err != nil {
		fmt.Println(err)
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

//listens to incoming client connections and binds a port for the publisher
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

func SetEncryptionKey(enckey string) {
	key = []byte(enckey)
}
