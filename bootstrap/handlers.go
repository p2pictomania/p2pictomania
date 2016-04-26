package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bogdanovich/dns_resolver"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

type addPlayerToRoomJSON struct {
	RoomID         int    `json:"roomID"`
	PlayerNickName string `json:"nickName"`
	PlayerIP       string `json:"playerIP"`
	RoomName       string `json:"roomName"`
}

type deletePlayerFromRoomJSON struct {
	RoomID         int    `json:"roomID"`
	PlayerNickName string `json:"nickName"`
}

// type newPlayerJSON struct {
// 	Name   string `json:"nickname"`
// 	IP     string `json:"ip"`
// 	Active bool   `json:"active"`
// }

//Index handler handles the landing page of the UI
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi")
}

// GetPeersForRoom handler retuns the peers for a given room name
func GetPeersForRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	urlVars := mux.Vars(r)
	log.Printf("Peers for room %s requested", urlVars["roomid"])
	roomID := urlVars["roomid"]
	query := "SELECT * from player_room_mapping where room_id = " + roomID
	result, err := sqlQuery(query)
	if err != nil {
		log.Println("Couldn't fetch players in room")
		http.Error(w, "Couldn't fetch players in room", http.StatusInternalServerError)
		return
	}

	jsonData := result.(map[string]interface{})
	results := jsonData["results"].([]interface{})
	row := results[0].(map[string]interface{})

	// TODO: will throw an error if there are no rooms with the given name.. code is fragile af !!
	values := row["values"].([]interface{})
	json.NewEncoder(w).Encode(values)
}

// AddPlayerToRoom handler adds the given player to a given room in the db
func AddPlayerToRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	decoder := json.NewDecoder(r.Body)
	var j addPlayerToRoomJSON
	err := decoder.Decode(&j)
	if err != nil {
		log.Println("Couldn't add player to room: " + err.Error())
		http.Error(w, "Couldn't add player to room", http.StatusInternalServerError)
		return
	}

	query := "INSERT into player_room_mapping values (" + strconv.Itoa(j.RoomID) + ", \"" + j.PlayerNickName + "\", \"" + j.PlayerIP + "\", \"" + j.RoomName + "\");"
	err = sqlExecute(query)
	if err != nil {
		log.Println("Couldn't add player to room - DB error: " + err.Error())
		http.Error(w, "Couldn't add player to room", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

// DeletePlayerFromRoom is used to delete a player from a room in the database
func DeletePlayerFromRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	decoder := json.NewDecoder(r.Body)
	var j deletePlayerFromRoomJSON
	err := decoder.Decode(&j)

	if err != nil {
		log.Println("Could not delete player from room")
		http.Error(w, "Could not delete player from room", http.StatusInternalServerError)
		return
	}

	query := "DELETE from player_room_mapping where room_id=" + strconv.Itoa(j.RoomID) + " and player_name= \"" + j.PlayerNickName + "\";"
	log.Println("Delete query:" + string(query))
	err = sqlExecute(query)

	if err != nil {
		log.Println("Could not delete player from room - DB error")
		http.Error(w, "Could not delete player from room", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

// DeletePlayerFromNetwork deletes a player from all rooms
func DeletePlayerFromNetwork(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	urlVars := mux.Vars(r)
	nickname := urlVars["nickname"]
	query := "DELETE from player_room_mapping where player_name = \"" + nickname + "\""
	_, err := sqlQuery(query)
	if err != nil {
		log.Println("Couldn't delete in room")
		http.Error(w, "Couldn't delete player in room", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

// AddNewPlayer adds a new player into the database
func AddNewPlayer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.ParseForm()

	name := r.FormValue("nickname")
	ip := r.FormValue("ip")
	if name == "" || ip == "" {
		log.Println("Could not decode add player json")
		http.Error(w, "Could not decode add player json", http.StatusInternalServerError)
		return
	}

	query := "INSERT into users values (\"" + name + "\", \"" + ip + "\", 1);"
	log.Println("New Player Query:" + query)
	err := sqlExecute(query)
	if err != nil {
		log.Println("Failed Login, choose another Nickname")
		http.Error(w, "Failed Login, choose another Nickname", http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

// DeletePlayer removes a player from the DB
func DeletePlayer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	urlVars := mux.Vars(r)
	log.Printf("Deleting user %s", urlVars["nickname"])
	nickname := urlVars["nickname"]

	query := "DELETE from users where name=\"" + nickname + "\";"
	err := sqlExecute(query)
	if err != nil {
		log.Println("Failed Logout/Delete player from DB")
		http.Error(w, "Failed Logout/Delete player from DB", http.StatusUnauthorized)
		return
	}
	log.Println("Successfully deleted player from DB")
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

// CreateNewRoom is the handler to create a new room
func CreateNewRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.ParseForm()

	roomname := r.FormValue("roomname")
	if roomname == "" {
		log.Println("Could not create new room")
		http.Error(w, "Could not create new room", http.StatusInternalServerError)
		return
	}

	query := "INSERT into rooms(name, open) values (\"" + roomname + "\", 0);"
	log.Println("New room Query:" + query)
	err := sqlExecute(query)
	if err != nil {
		log.Println("Failed creating a room, choose another roomname")
		http.Error(w, "Failed creating a room, choose another roomname", http.StatusUnauthorized)
		return
	}
	query = "SELECT id from rooms where name = \"" + roomname + "\";"
	data, _ := sqlQuery(query)
	jsonData := data.(map[string]interface{})
	results := jsonData["results"].([]interface{})
	row := results[0].(map[string]interface{})

	// TODO: will throw an error if there are no rooms with the given name.. code is fragile af !!
	values := row["values"].([]interface{})
	roomID := values[0].([]interface{})
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK, "roomID": int(roomID[0].(float64))})
}

// GetRoomsList is he handler to return the current list of rooms
func GetRoomsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	log.Println("Query for room list")
	query := "SELECT * from rooms where open = 1;"
	result, err := sqlQuery(query)
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

// GetRoomsListForPlayer is he handler to return the current list of rooms
func GetRoomsListForPlayer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	log.Println("Query for room list for specific player")
	urlVars := mux.Vars(r)
	nickname := urlVars["nickname"]

	query := "SELECT room_id, room_name from player_room_mapping where player_name = \"" + nickname + "\";"
	result, err := sqlQuery(query)
	if err != nil {
		log.Println("Couldn't fetch room list for specific player")
		http.Error(w, "Couldn't fetch room list for specific player", http.StatusInternalServerError)
		return
	}
	jsonData := result.(map[string]interface{})
	results := jsonData["results"].([]interface{})
	row := results[0].(map[string]interface{})
	json.NewEncoder(w).Encode(row)
}

// OpenRoom is the handler to change a room's open state to 1
func OpenRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	urlVars := mux.Vars(r)
	roomID := urlVars["roomID"]

	query := "UPDATE rooms SET open = 1 where id= " + roomID + ";"
	err := sqlExecute(query)
	if err != nil {
		log.Println("Failed to update room to Open: " + err.Error())
		http.Error(w, "Failed to update room to Open", http.StatusInternalServerError)
		return
	}
	log.Println("Successfully Opened room " + roomID)
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

// CloseRoom closes a given room with id roomID
func CloseRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	urlVars := mux.Vars(r)
	roomID := urlVars["roomID"]

	query := "UPDATE rooms SET open = 0 where id= " + roomID + ";"
	err := sqlExecute(query)
	if err != nil {
		log.Println("Failed to update room to closed: " + err.Error())
		http.Error(w, "Failed to update room to closed", http.StatusInternalServerError)
		return
	}
	log.Println("Successfully Opened room " + roomID)
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})

}

func sqlQuery(query string) (interface{}, error) {
	listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)
	leaderIP, err := GetLeaderIP(listOfBootstrapNodes)
	if err != nil {
		return nil, errors.New("No Leader found to execute query")
	}

	u := "http://" + leaderIP + ":" + strconv.Itoa(DBApiPort) + "/" + "db/query?pretty&timings"
	// u := "http://" + Config.DNS + ":" + strconv.Itoa(DBApiPort) + "/" + "db/query?pretty&timings"
	var endpoint *url.URL
	endpoint, err = url.Parse(u)
	parameters := url.Values{}
	parameters.Add("q", query)
	endpoint.RawQuery = parameters.Encode()

	resp, err := http.Get(endpoint.String())
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}
	defer resp.Body.Close()

	// check for execution response
	content, _ := ioutil.ReadAll(resp.Body)
	var j interface{}
	err = json.Unmarshal(content, &j)
	if err != nil {
		log.Printf("Could not read json response from db server: %s", err)
		return nil, err
	}
	data := j.(map[string]interface{})
	results := data["results"].([]interface{})
	for _, result := range results {
		output := result.(map[string]interface{})
		if _, ok := output["error"]; ok {
			err = errors.New("Could not execute query")
			log.Printf("Could not execute query %s : %s", query, err)
			return nil, err
		}
	}
	return data, nil
}

func sqlExecute(query string) error {
	//listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)

	resolver := dns_resolver.New([]string{"ns1.dnsimple.com", "ns2.dnsimple.com"})
	// In case of i/o timeout
	resolver.RetryTimes = 5

	bootiplist, err := resolver.LookupHost(Config.DNS)

	if err != nil {
		log.Println("DNS lookup error for autogra.de in CheckForBootstrapNode")
		log.Fatal(err.Error())
	}

	listOfBootstrapNodes := []string{}

	for _, val := range bootiplist {
		listOfBootstrapNodes = append(listOfBootstrapNodes, val.String())
	}

	leaderIP, err := GetLeaderIP(listOfBootstrapNodes)

	if err != nil {
		return errors.New("No Leader found to execute query")
	}

	url := "http://" + leaderIP + ":" + strconv.Itoa(DBApiPort) + "/" + "db/execute?pretty&timings"
	jsonStr, _ := json.Marshal([]string{query})
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Close = true
	if err != nil {
		log.Printf("%s", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Cannot execute query %s: %s", query, err)
		return err
	}
	defer resp.Body.Close()

	// check for execution response
	content, _ := ioutil.ReadAll(resp.Body)
	var j interface{}
	err = json.Unmarshal(content, &j)
	if err != nil {
		log.Printf("Could not read json response from db server: %s", err)
		return err
	}
	data := j.(map[string]interface{})
	results := data["results"].([]interface{})
	for _, result := range results {
		output := result.(map[string]interface{})
		if _, ok := output["error"]; ok {
			err = errors.New("Could not execute query")
			log.Printf("Could not execute query %s : %s", query, err)
			return err
		}
	}
	return nil
}

// RemoveBootstrapPeer removes node from raft group
func RemoveBootstrapPeer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	urlVars := mux.Vars(r)
	ip := urlVars["ip"]
	listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)
	if contains(listOfBootstrapNodes, ip) {
		leaderIP, err := GetLeaderIP(listOfBootstrapNodes)
		if err != nil {
			http.Error(w, "Failed to delete bootstrap node", http.StatusInternalServerError)
		}
		publicIP, err := GetPublicIP()
		if err != nil {
			http.Error(w, "Failed to delete bootstrap node", http.StatusInternalServerError)
		}
		if leaderIP != publicIP {
			u := "http://" + leaderIP + ":" + strconv.Itoa(BootstrapPort) + "/bootstrap/remove/" + ip
			_, err := http.Get(u)
			if err != nil {
				log.Printf("%s", err)
				http.Error(w, "Failed to delete bootstrap node", http.StatusInternalServerError)
			}
		} else {
			u := "http://localhost:+" + strconv.Itoa(DBApiPort) + "/removepeer?ip=" + ip
			_, err := http.Get(u)
			if err != nil {
				log.Printf("%s", err)
				http.Error(w, "Failed to delete bootstrap node", http.StatusInternalServerError)
			}
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
