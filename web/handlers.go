package web

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/flosch/pongo2"
	"github.com/gorilla/mux"
	sql "github.com/otoolep/rqlite/db"
	httpd "github.com/otoolep/rqlite/http"
	"github.com/otoolep/rqlite/store"
	"github.com/p2pictomania/p2pictomania/game"
)

var tplIndex = pongo2.Must(pongo2.FromFile("web/templates/index.html"))
var tplLogin = pongo2.Must(pongo2.FromFile("web/templates/login.html"))
var tplRooms = pongo2.Must(pongo2.FromFile("web/templates/rooms.html"))
var tplDraw = pongo2.Must(pongo2.FromFile("web/templates/draw.html"))
var tplGame = pongo2.Must(pongo2.FromFile("web/templates/game.html"))

// GameStore is the global for the room store
var GameStore *store.Store

var quit = make(chan bool, 1)
var done = make(chan bool, 1)

type setRoundForRoom struct {
	RoundID int `json:"roundID"`
	RoomID  int `json:"roomID"`
}

type selectWordForRound struct {
	Word     string `json:"word"`
	RoundID  int    `json:"roundID"`
	RoomID   int    `json:"roomID"`
	NickName string `json:"nickName"`
}

type roundReadyResults struct {
	Columns []string        `json:"columns"`
	Time    float64         `json:"time"`
	Types   []string        `json:"types"`
	Values  [][]interface{} `json:"values"`
}

type resultStruct struct {
	Result string `json:"result"`
}

// httpError returns a HTTP 5xx error
func httpError(err error, w http.ResponseWriter) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//Index handler handles the landing page of the UI
func Index(w http.ResponseWriter, r *http.Request) {
	err := tplIndex.ExecuteWriter(pongo2.Context{"testValue": "Hello World"}, w)
	httpError(err, w)
}

//Login handler handles the login page of the UI
func Login(w http.ResponseWriter, r *http.Request) {
	ip, _ := GetPublicIP()
	err := tplLogin.ExecuteWriter(pongo2.Context{"dns": Config.BootstrapDNSEndpoint, "ip": ip, "nickname": Nickname}, w)
	httpError(err, w)
}

// AuthUser is used to set the current user
func AuthUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.ParseForm()

	name := r.FormValue("nickname")
	if name == "" {
		http.Error(w, "nickname not loggedin", http.StatusInternalServerError)
		return
	}
	Nickname = name
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

func SetRoundForRoom(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var j setRoundForRoom
	err := decoder.Decode(&j)
	if err != nil {
		log.Println("Could not set round and room")
		http.Error(w, "Could not set round and room", http.StatusInternalServerError)
		return
	}

	leaderIP, err := game.GetRoomLeader(j.RoomID)

	query := "INSERT into round_room_mapping values (" + strconv.Itoa(j.RoundID) + ", " + strconv.Itoa(j.RoomID) + ");"
	err = game.SqlExecute(query, leaderIP)

	if err != nil {
		log.Println("Could not set round and room - DB error")
		http.Error(w, "Could not set round and room - DB error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

func random(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}

func GetWords(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		log.Println("Invalid room id passed")
		http.Error(w, "Invalid room id passed", http.StatusInternalServerError)
		return
	}

	num := r.Form.Get("num")
	numint, err := strconv.Atoi(num)

	if err != nil {
		log.Println("Invalid num parameter")
		http.Error(w, "Invalid num parameter", http.StatusInternalServerError)
		return
	}

	log.Printf("Get %s words requested", num)

	//TODO: move to constants
	var path string = "game/words.txt"

	file, err := os.Open(path)
	if err != nil {
		log.Println("Error while opening game/words.txt")
		http.Error(w, "Error while opening game/words.txt", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	log.Println(lines)

	//var words []string
	var buffer bytes.Buffer

	var i int = 0
	for ; i < numint; i++ {
		myrand := random(0, len(lines))
		//words = append(words, lines[myrand])
		buffer.WriteString(lines[myrand])
		if i == numint-1 {
			continue
		} else {
			buffer.WriteString(" ")
		}

	}

	res := resultStruct{Result: buffer.String()}

	//TODO: return num random words from words.txt
	json.NewEncoder(w).Encode(res)
}

func GetRoundForRoom(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		log.Println("Invalid room id passed")
		http.Error(w, "Invalid room id passed", http.StatusInternalServerError)
		return
	}

	roomID := r.Form.Get("roomid")
	log.Printf("GetRoundForRoom %s requested", roomID)

	roomIDint, err := strconv.Atoi(roomID)

	leaderIP, err := game.GetRoomLeader(roomIDint)

	if err != nil {
		log.Println("Error while getting room leader")
		http.Error(w, "Error while getting room leader", http.StatusInternalServerError)
		return
	}

	query := "SELECT * from round_room_mapping where room_id=" + roomID + ";"
	result, err := game.SqlQuery(query, leaderIP)

	if err != nil {
		log.Println("Couldn't fetch room list")
		http.Error(w, "Couldn't fetch room list", http.StatusInternalServerError)
		return
	}

	jsonData := result.(map[string]interface{})
	results := jsonData["results"].([]interface{})
	row := results[0].(map[string]interface{})
	json.NewEncoder(w).Encode(row)
}

func SelectWordForRound(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var j selectWordForRound
	err := decoder.Decode(&j)
	if err != nil {
		log.Println("Could not select word for round")
		http.Error(w, "Could not select word for round", http.StatusInternalServerError)
		return
	}

	leaderIP, err := game.GetRoomLeader(j.RoomID)

	query := "INSERT into words_round_mapping values (" + strconv.Itoa(j.RoundID) + ", " + strconv.Itoa(j.RoomID) + ", \"" + j.NickName + "\", \"" + j.Word + "\");"
	err = game.SqlExecute(query, leaderIP)

	if err != nil {
		log.Println("Could not select word for round - DB error")
		http.Error(w, "Could not select word for round - DB error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

func CheckGuess(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		log.Println("Invalid request")
		http.Error(w, "Invalid request", http.StatusInternalServerError)
		return
	}

	roomID := r.Form.Get("roomid")
	roundID := r.Form.Get("roundid")
	guess := r.Form.Get("guess")
	drawer := r.Form.Get("drawer")

	log.Printf("Check guess %s - %s requested", roundID, guess)

	roomIDint, err := strconv.Atoi(roomID)

	leaderIP, err := game.GetRoomLeader(roomIDint)

	if err != nil {
		log.Println("Error while getting room leader")
		http.Error(w, "Error while getting room leader", http.StatusInternalServerError)
		return
	}

	query := "SELECT word from words_round_mapping where room_id=" + roomID + " and round_id=" + roundID + " and player_name=\"" + drawer + "\";"
	result, err := game.SqlQuery(query, leaderIP)

	if err != nil {
		log.Println("Could not check guess from db")
		http.Error(w, "Could not check guess from db", http.StatusInternalServerError)
		return
	}

	jsonData := result.(map[string]interface{})
	results := jsonData["results"].([]interface{})
	row := results[0].(map[string]interface{})
	valuesArr := row["values"].([]interface{})
	valueRow := valuesArr[0].([]interface{})

	//TODO: type assertion needs fixing
	var value string = (valueRow[0]).(string)

	if value == guess {
		res := resultStruct{Result: "true"}
		json.NewEncoder(w).Encode(res)
	} else {
		res := resultStruct{Result: "false"}
		json.NewEncoder(w).Encode(res)
	}

}

func GetScore(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		log.Println("Invalid request")
		http.Error(w, "Invalid request", http.StatusInternalServerError)
		return
	}

	roomID := r.Form.Get("roomid")
	drawer := r.Form.Get("drawer")

	log.Printf("Get Score for %s - %s requested", drawer, roomID)

	roomIDint, err := strconv.Atoi(roomID)

	leaderIP, err := game.GetRoomLeader(roomIDint)

	if err != nil {
		log.Println("Error while getting room leader")
		http.Error(w, "Error while getting room leader", http.StatusInternalServerError)
		return
	}

	query := "SELECT score from player_score_mapping where room_id=" + roomID + " and player_name=\"" + drawer + "\";"
	result, err := game.SqlQuery(query, leaderIP)

	if err != nil {
		log.Println("Could not check guess from db")
		http.Error(w, "Could not check guess from db", http.StatusInternalServerError)
		return
	}

	jsonData := result.(map[string]interface{})
	results := jsonData["results"].([]interface{})
	row := results[0].(map[string]interface{})

	valuesArr := row["values"].([]interface{})
	valueRow := valuesArr[0].([]interface{})

	//TODO: type assertion needs fixing
	var value float64 = (valueRow[0]).(float64)
	var intvalue = int(value)
	var stringvalue = strconv.Itoa(intvalue)
	res := resultStruct{Result: stringvalue}
	json.NewEncoder(w).Encode(res)

}

func SetScore(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		log.Println("Invalid request")
		http.Error(w, "Invalid request", http.StatusInternalServerError)
		return
	}

	roomID := r.Form.Get("roomid")
	drawer := r.Form.Get("drawer")
	score := r.Form.Get("score")

	log.Printf("Set Score for %s - %s requested", drawer, roomID)

	roomIDint, err := strconv.Atoi(roomID)

	leaderIP, err := game.GetRoomLeader(roomIDint)

	if err != nil {
		log.Println("Error while getting room leader")
		http.Error(w, "Error while getting room leader", http.StatusInternalServerError)
		return
	}

	query := "INSERT into player_score_mapping values (" + roomID + ", \"" + drawer + "\", " + score + ");"
	err = game.SqlExecute(query, leaderIP)

	if err != nil {
		log.Println("Could not set round and room - DB error")
		http.Error(w, "Could not set round and room - DB error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})

}

/*
func UpdateScore(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		log.Println("Invalid request")
		http.Error(w, "Invalid request", http.StatusInternalServerError)
		return
	}

	roomID := r.Form.Get("roomid")
	drawer := r.Form.Get("drawer")
	guessor := r.Form.Get("guessor")
	log.Printf("Update Score for %s requested", roomID)

	roomIDint, err := strconv.Atoi(roomID)

	leaderIP, err := game.GetRoomLeader(roomIDint)

	if err != nil {
		log.Println("Error while getting room leader")
		http.Error(w, "Error while getting room leader", http.StatusInternalServerError)
		return
	}

	query := "UPDATE player_score_mapping SET score = score + 10 where room_id=" + roomID + " and (nick=\"" + drawer + "\" or nick=\"" + guessor + "\");"
	result, err := game.SqlQuery(query, leaderIP)
	_ = result
	if err != nil {
		log.Println("Could not update score in db")
		http.Error(w, "Could not update score db", http.StatusInternalServerError)
		return
	}

	//	jsonData := result.(map[string]interface{})
	//	results := jsonData["results"].([]interface{})
	//	row := results[0].(map[string]interface{})

	//TODO: return status 200

	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}
*/

func IsRoundReady(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		log.Println("Unable to parse request")
		http.Error(w, "Unable to parse request", http.StatusInternalServerError)
		return
	}

	roomID := r.Form.Get("roomid")
	roundID := r.Form.Get("roundid")
	num_members := r.Form.Get("num")
	num_members_int, err := strconv.Atoi(num_members)

	if err != nil {
		log.Println("Unable to parse num")
		http.Error(w, "Unable to parse num", http.StatusInternalServerError)
		return
	}

	log.Printf("RoundReady check for %s requested", roomID)

	roomIDint, err := strconv.Atoi(roomID)

	leaderIP, err := game.GetRoomLeader(roomIDint)

	if err != nil {
		log.Println("Error while getting room leader")
		http.Error(w, "Error while getting room leader", http.StatusInternalServerError)
		return
	}

	query := "SELECT COUNT(*) from words_round_mapping where room_id=" + roomID + " and round_id=" + roundID + ";"
	result, err := game.SqlQuery(query, leaderIP)

	if err != nil {
		log.Println("Could not check for round readiness")
		http.Error(w, "Could not check for round readiness", http.StatusInternalServerError)
		return
	}

	jsonData := result.(map[string]interface{})
	results := jsonData["results"].([]interface{})
	row := results[0].(map[string]interface{})
	valuesArr := row["values"].([]interface{})
	valueRow := valuesArr[0].([]interface{})

	//TODO: type assertion needs fixing
	var value float64 = (valueRow[0]).(float64)

	//var value int = 0

	if int(value) == num_members_int {
		res := resultStruct{Result: "true"}
		json.NewEncoder(w).Encode(res)

	} else {
		//return false
		res := resultStruct{Result: "false"}
		json.NewEncoder(w).Encode(res)
	}

}

// Logout is used to set the current user
func Logout(w http.ResponseWriter, r *http.Request) {
	if Nickname == "" {
		Login(w, r)
		return
	}
	url := Config.BootstrapDNSEndpoint + "/player/delete/" + Nickname
	log.Println("Delete url: " + url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "nickname not logged out", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("nickname not logged out: %s", resp.Status)
		http.Error(w, "nickname not logged out", http.StatusInternalServerError)
		return
	}
	Nickname = ""
	Login(w, r)
}

// RoomList returns a page with the list of rooms that are available to join
func RoomList(w http.ResponseWriter, r *http.Request) {
	if Nickname == "" {
		Login(w, r)
		return
	}
	ip, _ := GetPublicIP()
	err := tplRooms.ExecuteWriter(pongo2.Context{"nickname": Nickname, "dns": Config.BootstrapDNSEndpoint, "playerIP": ip}, w)
	httpError(err, w)
}

// Game handler handles the landing page of the UI
func Game(w http.ResponseWriter, r *http.Request) {
	if Nickname == "" {
		Login(w, r)
		return
	}
	urlVars := mux.Vars(r)
	roomID := urlVars["roomID"]
	ip, _ := GetPublicIP()
	var err error
	var listOfPlayers []interface{}
	var leaderIP string

	// Start concensus if room has only one player
	listOfPlayers, err = getListOfPlayersForRoom(roomID)
	if err != nil {
		httpError(err, w)
		return
	}
	if len(listOfPlayers) == 1 {
		go setupGameDB("")
		markRoomAsOpen(roomID)
	} else {
		listOfIPs := getListOfIPs(listOfPlayers)
		leaderIP, err = getLeaderIP(listOfIPs)
		if err != nil {
			httpError(err, w)
			return
		}
		go setupGameDB(leaderIP)
	}

	err = tplGame.ExecuteWriter(pongo2.Context{"nickname": Nickname,
		"dns": Config.BootstrapDNSEndpoint, "roomID": roomID,
		"maxPlayers": MaxRoomPlayers, "playerIP": ip}, w)
	httpError(err, w)
}

// HandleSocketConn is used as the endpoint fot websocket connections to be made
func HandleSocketConn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	c := &connection{send: make(chan []byte, 256), ws: ws}
	Hub.register <- c
	go c.WriteMessagesToSocket()
	c.ReadMessagesFromSocket()
}

func getListOfIPs(list []interface{}) []string {
	var ips []string
	for _, val := range list {
		entry := val.([]interface{})
		ip := entry[2].(string)
		ips = append(ips, ip)
	}
	return ips
}

func markRoomAsOpen(roomID string) error {
	url := Config.BootstrapDNSEndpoint + "/room/open/" + roomID
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Could not mark room as open: " + err.Error())
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Println("Could not mark room as open")
		return errors.New("Could not open room")
	}
	return nil
}

func getListOfPlayersForRoom(roomID string) ([]interface{}, error) {
	url := Config.BootstrapDNSEndpoint + "/peers/" + roomID
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Could not get peers")
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var j interface{}
	err = json.Unmarshal(contents, &j)
	if err != nil {
		return nil, err
	}
	data := j.([]interface{})
	return data, nil
}

func dbExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func getPublicIP() (string, error) {
	resp, err := http.Get("http://ipv4.icanhazip.com/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ip)), nil
}

func join(joinAddr, raftAddr string) error {
	publicIP, _ := getPublicIP()
	b, err := json.Marshal(map[string]string{"addr": publicIP + ":" + strconv.Itoa(GameDBRaftPort)})
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s:%d/join", joinAddr, GameDBApiPort), "application-type/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func waitForAPIStartAndLeader() {
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/status", GameDBApiPort)
	res, err := client.Get(url)
	if err != nil {
		log.Fatalf("Could not reach api server - Timed out : %s", err)
	}
	//TODO: check leader status instead of waiting 5 seconds
	time.Sleep(5000 * time.Millisecond)
	defer res.Body.Close()
}

func getLeaderIP(listOfNodes []string) (string, error) {
	for _, ip := range listOfNodes {
		url := fmt.Sprintf("http://%s:%d/status", ip, GameDBApiPort)
		res, err := http.Get(url)
		if err != nil {
			log.Println(err)
			continue
		}
		defer res.Body.Close()
		content, err := ioutil.ReadAll(res.Body)
		var j interface{}
		err = json.Unmarshal(content, &j)
		if err != nil {
			log.Println(err)
			continue
		}
		data := j.(map[string]interface{})
		store := data["store"].(map[string]interface{})
		raft := store["raft"].(map[string]interface{})
		state := raft["state"].(string)
		if state == "Leader" {
			return ip, nil
		}
	}
	return "", errors.New("Could not find bootstrap hosts")
}

func setupGameDB(joinAddr string) {
	log.Println("setting up db")
	log.Println(GameStore)
	if GameStore != nil {
		log.Println("Trying to clean up DB")
		quit <- true
		cleanup := <-done
		if cleanup {
			log.Println("Successful clean up DB")
		}
	}
	if dbExists(GameDBFolder) {
		os.RemoveAll(GameDBFolder)
	}
	dataPath := GameDBFolder
	httpAddr := ":" + strconv.Itoa(GameDBApiPort)
	raftAddr := ":" + strconv.Itoa(GameDBRaftPort)
	disRedirect := true
	dataPath, err := filepath.Abs(dataPath)
	if err != nil {
		log.Fatalf("failed to determine absolute data path: %s", err.Error())
	}
	dbConf := sql.NewConfig()
	dbConf.DSN = ""
	dbConf.Memory = false
	GameStore = store.New(dbConf, dataPath, raftAddr)
	log.Println("set GameStore: ")
	log.Println(GameStore)
	if err := GameStore.Open(joinAddr == ""); err != nil {
		log.Fatalf("failed to open store: %s", err.Error())
	}

	// If join was specified, make the join request.
	if joinAddr != "" {
		if err := join(joinAddr, raftAddr); err != nil {
			log.Fatalf("failed to join node at %s: %s", joinAddr, err.Error())
		}
	}

	// Create the HTTP query server.
	s := httpd.New(httpAddr, GameStore)
	s.DisableRedirect = disRedirect
	if err := s.Start(); err != nil {
		log.Fatalf("failed to start HTTP server: %s", err.Error())

	}

	if joinAddr == "" {
		// if fresh DB.. initialize all tables
		waitForAPIStartAndLeader()
		game.InitTables()
	}

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)

	for {
		select {
		case <-quit:
			if err := GameStore.Close(); err != nil {
				log.Printf("failed to close store: %s", err.Error())
			}
			log.Println("Closed raft")
			s.Close()
			log.Println("Closed http")
			time.Sleep(time.Second * 3)
			log.Println("sleep over")
			GameStore = nil
			done <- true
			return
		case <-terminate:
			if err := GameStore.Close(); err != nil {
				log.Printf("failed to close store: %s", err.Error())
			}
			s.Close()
			log.Println("rqlite server stopped")
			os.Exit(0)
		}
	}
}
