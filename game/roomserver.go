package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config object stores the values in the config.json file
var Config ConfigObject

//ConfigObject holds the parsed config.json file
type ConfigObject struct {
	DNS               string `json:"dns"`
	DnsimpleURL       string `json:"dnsimpleURL"`
	DnsimpleAuthToken string `json:"dnsimpleAuthToken"`
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

func parseConfigFile() (configObject ConfigObject) {
	configFile, error := ioutil.ReadFile(ConfigFileName)
	if error != nil {
		log.Fatalf("Config file not found: %s", error)
	}
	var config ConfigObject
	json.Unmarshal(configFile, &config)
	return config
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
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

// InitTables initializes tables
func InitTables() {
	query := "[" +
		"\"CREATE TABLE `round_room_mapping` (`round_id` INTEGER NOT NULL, `room_id` INTEGER NOT NULL, UNIQUE (`room_id`) ON CONFLICT REPLACE);\"," +
		"\"CREATE TABLE `words_round_mapping` (`round_id` INTEGER NOT NULL, `room_id` INTEGER NOT NULL, `player_name` TEXT NOT NULL, `word` TEXT NOT NULL, UNIQUE (`round_id`, `player_name`, `room_id`) ON CONFLICT REPLACE);\"," +
		"\"CREATE TABLE `player_score_mapping` (`room_id` INTEGER NOT NULL, `player_name` TEXT NOT NULL, `score` INTEGER, UNIQUE (`room_id`, `player_name`) ON CONFLICT REPLACE);\"," +
		"\"CREATE TABLE `round_room_end_mapping` (`round_id` INTEGER NOT NULL, `room_id` INTEGER NOT NULL, `player_name` TEXT NOT NULL, UNIQUE (`round_id`, `room_id`, `player_name`) ON CONFLICT REPLACE);\"" +
		"]"

	url := "http://localhost:" + strconv.Itoa(GameDBApiPort) + "/" + "db/execute?pretty&timings"
	var jsonStr = []byte(query)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Close = true
	if err != nil {
		log.Fatalf("%s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Cannot execute table creation queries: %s", err)
	}
	defer resp.Body.Close()

	// check for execution response
	content, err := ioutil.ReadAll(resp.Body)

	var j interface{}
	err = json.Unmarshal(content, &j)
	if err != nil {
		log.Fatalf("Could not read json response from game db server: %s", err)
	}
	data := j.(map[string]interface{})
	results := data["results"].([]interface{})
	for i, result := range results {
		output := result.(map[string]interface{})
		if val, ok := output["error"]; ok {
			log.Fatalf("Could not execute query %d in querySet %s : %s", i, query, val)
		}
	}
	log.Println("Room DB tables initialized")
}

// GetRoomLeader foo
func GetRoomLeader(roomID int) (string, error) {

	values, err := GetListOfPlayersForRoom(strconv.Itoa(roomID))
	if err != nil {
		log.Println("Could not get list of peers for room :" + err.Error())
		return "", err
	}
	log.Println("List of IPs for Room: ")
	log.Println(values)
	var listofRoomNodes []string
	for _, val := range values {
		row := val.([]interface{})
		listofRoomNodes = append(listofRoomNodes, row[2].(string))
	}
	numMembers := len(listofRoomNodes)

	if numMembers == 0 {
		return "", errors.New("no room members found")
	}

	log.Println("List of room nodes=")
	log.Println(listofRoomNodes)
	leaderIP, err := GetLeaderIP(listofRoomNodes)

	if err != nil {
		return "", errors.New("No Leader found to execute query")
	}

	return leaderIP, nil

}

// SqlQuery does shit
func SqlQuery(query string, leaderIP string) (interface{}, error) {
	//listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)

	u := "http://" + leaderIP + ":" + strconv.Itoa(GameDBApiPort) + "/" + "db/query?pretty&timings"
	// u := "http://" + Config.DNS + ":" + strconv.Itoa(DBApiPort) + "/" + "db/query?pretty&timings"
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
			err = errors.New("Could not execute query")
			log.Printf("Could not execute query %s : %s", query, err)
			return nil, err
		}
	}
	return data, nil
}

// SqlExecute does shit
func SqlExecute(query string, leaderIP string) error {
	//listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)

	url := "http://" + leaderIP + ":" + strconv.Itoa(GameDBApiPort) + "/" + "db/execute?pretty&timings"
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

// StartServer does shit
func StartServer() {
	Config = parseConfigFile()
	log.Printf("config=%+v\n", Config)
}
