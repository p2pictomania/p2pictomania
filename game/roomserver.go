package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	sql "github.com/otoolep/rqlite/db"
	httpd "github.com/otoolep/rqlite/http"
	"github.com/otoolep/rqlite/store"
	"github.com/p2pictomania/p2pictomania/connections"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
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

func initTables() {
	query := "[" +
		"\"CREATE TABLE `round_room_mapping` (`round_id` INTEGER NOT NULL, `room_id` INTEGER NOT NULL, UNIQUE (`room_id`, `round_id`) ON CONFLICT REPLACE);\"," +
		"\"CREATE TABLE `words_round_mapping` (`round_id` INTEGER NOT NULL, `room_id` INTEGER NOT NULL, `player_name` TEXT NOT NULL, `word` TEXT NOT NULL, UNIQUE (`round_id`, `player_name`, `word`) ON CONFLICT REPLACE);\"," +
		"\"CREATE TABLE `player_score_mapping` (`room_id` INTEGER NOT NULL, `player_name` TEXT NOT NULL, `score` INTEGER, UNIQUE (`room_id`, `player_name`) ON CONFLICT REPLACE);\"" +
		"]"

	url := "http://localhost:" + strconv.Itoa(DBApiPort) + "/" + "db/execute?pretty&timings"
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

	log.Println("Response from initTable=" + string(content))
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
}

func GetRoomLeader(roomID int) (string, error) {

	membersList, numMembers, errRoom := connections.QueryRoom(roomID)

	if errRoom != nil {
		return "", errRoom
	}

	if numMembers == 0 {
		return "", errors.New("no room members found")
	}

	listofRoomNodes := []string{}

	for i := 0; i < numMembers; i++ {
		listofRoomNodes = append(listofRoomNodes, membersList[i].IP)
	}

	log.Println("List of room nodes=")
	log.Println(listofRoomNodes)
	leaderIP, err := GetLeaderIP(listofRoomNodes)

	if err != nil {
		return "", errors.New("No Leader found to execute query")
	}

	return leaderIP, nil

}

func SqlQuery(query string, leaderIP string) (interface{}, error) {
	//listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)

	u := "http://" + leaderIP + ":" + strconv.Itoa(DBApiPort) + "/" + "db/query?pretty&timings"
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

func SqlExecute(query string, leaderIP string) error {
	//listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)

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

func join(joinAddr, raftAddr string) error {
	publicIP, _ := getPublicIP()
	b, err := json.Marshal(map[string]string{"addr": publicIP + ":" + strconv.Itoa(DBRaftPort)})
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s:%d/join", joinAddr, DBApiPort), "application-type/json", bytes.NewReader(b))
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
	url := fmt.Sprintf("http://127.0.0.1:%d/status", DBApiPort)
	res, err := client.Get(url)
	if err != nil {
		log.Fatalf("Could not reach api server - Timed out : %s", err)
	}
	//TODO: check leader status instead of waiting 5 seconds
	time.Sleep(5000 * time.Millisecond)
	defer res.Body.Close()
}

func SetupDB(joinAddr string) {
	if dbExists(DBFolder) {
		os.RemoveAll(DBFolder)
	}
	dataPath := DBFolder
	httpAddr := ":" + strconv.Itoa(DBApiPort)
	raftAddr := ":" + strconv.Itoa(DBRaftPort)
	disRedirect := true
	dataPath, err := filepath.Abs(dataPath)
	if err != nil {
		log.Fatalf("failed to determine absolute data path: %s", err.Error())
	}
	dbConf := sql.NewConfig()
	dbConf.DSN = ""
	dbConf.Memory = false
	store := store.New(dbConf, dataPath, raftAddr)
	if err := store.Open(joinAddr == ""); err != nil {
		log.Fatalf("failed to open store: %s", err.Error())
	}

	// If join was specified, make the join request.
	if joinAddr != "" {
		if err := join(joinAddr, raftAddr); err != nil {
			log.Fatalf("failed to join node at %s: %s", joinAddr, err.Error())
		}
	}

	// Create the HTTP query server.
	s := httpd.New(httpAddr, store)
	s.DisableRedirect = disRedirect
	if err := s.Start(); err != nil {
		log.Fatalf("failed to start HTTP server: %s", err.Error())

	}

	if joinAddr == "" {
		// if fresh DB.. initialize all tables
		waitForAPIStartAndLeader()
		initTables()
	}

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	if err := store.Close(); err != nil {
		log.Printf("failed to close store: %s", err.Error())
	}
	log.Println("rqlite server stopped")
	os.Exit(0)
}

func StartServer() {
	Config = parseConfigFile()
	log.Printf("config=%+v\n", Config)
}