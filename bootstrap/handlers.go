package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
)

type addPlayerToRoomJSON struct {
	RoomID         int    `json:"roomID"`
	PlayerNickName string `json:"nickName"`
	PlayerIP       string `json:"playerIP"`
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
	json.NewEncoder(w).Encode(string(result))
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

func sqlQuery(query string) ([]byte, error) {
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
	return content, nil
}

func sqlExecute(query string) error {
	listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)
	leaderIP, err := GetLeaderIP(listOfBootstrapNodes)

	if err != nil {
		return errors.New("No Leader found to execute query")
	}

	url := "http://" + leaderIP + ":" + strconv.Itoa(DBApiPort) + "/" + "db/execute?pretty&timings"
	jsonStr, _ := json.Marshal([]string{query})
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
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
