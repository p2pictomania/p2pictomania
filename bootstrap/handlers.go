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
	//"net"
	"net/http"
	"net/url"
	"strconv"
)

type addPlayerToRoomJSON struct {
	RoomID         int    `json:"roomID"`
	PlayerNickName string `json:"nickName"`
	PlayerIP       string `json:"playerIP"`
}

type deletePlayerFromRoomJSON struct {
	RoomID         int    `json:"roomID"`
	PlayerNickName string `json:"nickName"`
}

//Index handler handles the landing page of the UI
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi")
}

// GetPeersForRoom handler retuns the peers for a given room name
func GetPeersForRoom(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(result)
}

// AddPlayerToRoom handler adds the given player to a given room in the db
func AddPlayerToRoom(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var j addPlayerToRoomJSON
	err := decoder.Decode(&j)
	if err != nil {
		log.Println("Couldn't add player to room")
		http.Error(w, "Couldn't add player to room", http.StatusInternalServerError)
		return
	}

	query := "INSERT into player_room_mapping values (" + strconv.Itoa(j.RoomID) + ", \"" + j.PlayerNickName + "\", \"" + j.PlayerIP + "\");"
	err = sqlExecute(query)
	if err != nil {
		log.Println("Couldn't add player to room - DB error")
		http.Error(w, "Couldn't add player to room", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

func DeletePlayerFromRoom(w http.ResponseWriter, r *http.Request) {

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

func sqlQuery(query string) (interface{}, error) {
	// listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)
	// leaderIP, err := GetLeaderIP(listOfBootstrapNodes)
	//
	// if err != nil {
	// 	return nil, errors.New("No Leader found to execute query")
	// }

	// u := "http://" + leaderIP + ":" + strconv.Itoa(DBApiPort) + "/" + "db/query?pretty&timings"
	u := "http://" + Config.DNS + ":" + strconv.Itoa(DBApiPort) + "/" + "db/query?pretty&timings"
	var endpoint *url.URL
	endpoint, err := url.Parse(u)
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
			log.Printf("Could not execute query %s : %s", query, err)
			return err
		}
	}
	return nil
}
