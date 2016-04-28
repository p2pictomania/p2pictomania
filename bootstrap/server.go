package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sql "github.com/abhishekshivanna/rqlite/db"
	httpd "github.com/abhishekshivanna/rqlite/http"
	"github.com/abhishekshivanna/rqlite/store"
	"github.com/bogdanovich/dns_resolver"
)

// DNSRecord holds the data of a resolved DNS name
type DNSRecord []struct {
	Record struct {
		ID           int         `json:"id"`
		DomainID     int         `json:"domain_id"`
		ParentID     interface{} `json:"parent_id"`
		Name         string      `json:"name"`
		Content      string      `json:"content"`
		TTL          int         `json:"ttl"`
		Prio         interface{} `json:"prio"`
		RecordType   string      `json:"record_type"`
		SystemRecord bool        `json:"system_record"`
		CreatedAt    time.Time   `json:"created_at"`
		UpdatedAt    time.Time   `json:"updated_at"`
	} `json:"record"`
}

// Config object stores the values in the config.json file
var Config ConfigObject

//ConfigObject holds the parsed config.json file
type ConfigObject struct {
	DNS               string `json:"dns"`
	DnsimpleURL       string `json:"dnsimpleURL"`
	DnsimpleAuthToken string `json:"dnsimpleAuthToken"`
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

// GetPublicIP retunrs the public ip of the current node
func GetPublicIP() (string, error) {
	return getPublicIP()
}

func addSelfToDNS() error {
	url := Config.DnsimpleURL
	publicIP, err := getPublicIP()
	if err != nil {
		log.Fatalf("Cannot get public IP: %s", err)
		return err
	}
	var jsonStr = []byte(`{"record": {"name": "", "record_type": "A", "content": "` + publicIP + `", "ttl": 60, "prio": 10}}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Close = true
	req.Header.Set("X-DNSimple-Token", Config.DnsimpleAuthToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Cannot bind IP to DNS name: %s", err)
		return err
	}
	defer resp.Body.Close()
	// Failed creating DNS record
	if resp.StatusCode != http.StatusCreated {
		// if HTTP 400 check for error message returned
		if resp.StatusCode == http.StatusBadRequest {
			data, _ := ioutil.ReadAll(resp.Body)
			var j interface{}
			err = json.Unmarshal(data, &j)
			if err != nil {
				log.Fatalf("Could not read json response from DNSimple server: %s", err)
			}
			content := j.(map[string]interface{})
			message := content["message"].(string)
			suffix := "already exists so it was ignored."
			if strings.HasSuffix(strings.TrimSpace(message), suffix) {
				// We now know that this node has its IP in the DNS record
				// TODO: Handle nodes behine a NAT
				return nil
			}
		}
		return errors.New("DNS record could not be added")
	}
	return nil
}

// DeleteSelfFromDNS is used to delete an entry from the DNSimple records
func DeleteSelfFromDNS() {

	//list using curl  -H 'X-DNSimple-Token: <email>:<token>' \
	//-H 'Accept: application/json' \
	//https://api.dnsimple.com/v1/domains/example.com/records

	var listurl = "https://api.dnsimple.com/v1/domains/autogra.de/records?type=A"

	req, err := http.NewRequest("GET", listurl, nil)
	req.Close = true
	req.Header.Set("X-DNSimple-Token", Config.DnsimpleAuthToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Cannot bind IP to DNS name: %s", err)
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println("error reading response body to A record GET request")
		return
	}

	log.Println("Response Body=" + string(respBody))

	resultjson := DNSRecord{}
	json.Unmarshal(respBody, &resultjson)

	log.Printf("%+v", resultjson)
	log.Println(len(resultjson))

	publicIP, _ := getPublicIP()

	for _, val := range resultjson {

		//if NodeIP is found, get the "id" to delete
		if val.Record.Content == publicIP {
			fmt.Println(val.Record.ID)

			//delete the id with the following call
			//curl  -H 'X-DNSimple-Token: <email>:<token>' \
			//-H 'Accept: application/json' \
			//-H 'Content-Type: application/json' \
			//-X DELETE \
			//https: //api.dnsimple.com/v1/domains/example.com/records/2

			var deleteurl = "https://api.dnsimple.com/v1/domains/autogra.de/records/" + strconv.Itoa(val.Record.ID)

			delReq, err := http.NewRequest("DELETE", deleteurl, nil)
			delReq.Close = true
			delReq.Header.Set("X-DNSimple-Token", Config.DnsimpleAuthToken)
			delReq.Header.Set("Accept", "application/json")
			delReq.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			delResp, err := client.Do(delReq)
			if err != nil {
				log.Fatalf("Cannot bind IP to DNS name: %s", err)
				return
			}
			defer delResp.Body.Close()
			delRespBody, err := ioutil.ReadAll(delResp.Body)

			if err != nil {
				log.Println("error reading response body in DeleteSelfFromDNS")
				return
			}

			fmt.Println("Delete Response Body=" + string(delRespBody))

			//return after deleting 1 matching IP from DNS record
			return
		}
	}
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

func setupDB(joinAddr string) {
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
	publicIP, _ := getPublicIP()
	publicRaftAddr := fmt.Sprintf("%s%s", publicIP, raftAddr)
	if err := store.Open(joinAddr == "", publicRaftAddr); err != nil {
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
	time.Sleep(3000 * time.Millisecond)
	if err := store.Close(); err != nil {
		log.Printf("failed to close store: %s", err.Error())
	}
	s.Close()
	DeleteSelfFromDNS()
	if dbExists(DBFolder) {
		os.RemoveAll(DBFolder)
	}
	log.Println("bs db server stopped")
	os.Exit(0)
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

func initTables() {
	query := "[" +
		"\"CREATE TABLE IF NOT EXISTS `bootstrap` (`ip` TEXT, `active` INTEGER DEFAULT 1, PRIMARY KEY(ip));\"," +
		"\"CREATE TABLE `users` (`name` TEXT NOT NULL, `ip` TEXT NOT NULL, `active` INTEGER DEFAULT 1, PRIMARY KEY(name));\"," +
		"\"CREATE TABLE `rooms` (`id` INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, `name` TEXT NOT NULL, `open` INTEGER DEFAULT 1, UNIQUE (`name`));\"," +
		"\"CREATE TABLE `player_room_mapping` (`room_id` INTEGER NOT NULL, `player_name` TEXT NOT NULL, `player_ip` TEXT NOT NULL, `room_name` TEXT NOT NULL, UNIQUE (`room_id`, `player_name`, `room_name`) ON CONFLICT REPLACE);\"" +
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
	var j interface{}
	err = json.Unmarshal(content, &j)
	if err != nil {
		log.Fatalf("Could not read json response from db server: %s", err)
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

// checkForBootstrapNodes checks for existing Bootstrap nodes
// and returns true if if added itself to the Bootstrap nodes list
func checkForBootstrapNodes() bool {

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

	fmt.Print("Bootstrap result=")
	fmt.Println(listOfBootstrapNodes)

	//listOfBootstrapNodes, _ := net.LookupHost(Config.DNS)
	log.Printf("Detected %d Bootstrap Server(s) in the network", len(listOfBootstrapNodes))
	// Case where there are no Bootstrap nodes
	if len(listOfBootstrapNodes) == 0 {
		log.Println("No nodes bound to DNS name, setting up new bootstrap DB")
		err := addSelfToDNS()
		if err != nil {
			log.Println(err)
			return false
		}
		go setupDB("")
		return true
	} else if len(listOfBootstrapNodes) < MaxNumBootstrapNode {
		err := addSelfToDNS()
		if err != nil {
			log.Println(err)
			return false
		}
		leaderIP, err := GetLeaderIP(listOfBootstrapNodes)
		// If no leaderIP found assume all nodes in the bootstrap to be dead
		// bind self
		if err != nil {
			log.Printf("No leader IP found : %s", err)
			go setupDB("")
			return true
		}
		go setupDB(leaderIP)
		return true
	} else {
		// check to see if we were bootstrap node before (DNS bound).. if yes.. join the cluster
		publicIP, err := getPublicIP()
		if err != nil {
			log.Println("Public IP could not be fetched")
			return false
		}
		if stringInSlice(publicIP, listOfBootstrapNodes) {
			leaderIP, err := GetLeaderIP(listOfBootstrapNodes)
			if err != nil {
				log.Println(err)
				go setupDB("")
				return true
			}
			go setupDB(leaderIP)
			return true
		}
	}
	return false
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

//StartServer is the function used to start the local server used to
//interact with the web interface
func StartServer() {
	Config = parseConfigFile()
	iAmBootstrapNode := checkForBootstrapNodes()
	if !iAmBootstrapNode {
		log.Println("Not starting bootstrap endpoint")
		return
	}
	router := NewRouter()
	log.Println("Starting Bootstrap server on port " + strconv.Itoa(BootstrapPort))
	err := http.ListenAndServe(":"+strconv.Itoa(BootstrapPort), router)
	if err != nil {
		log.Fatalf("Failed to start the UI server: %s", err)
	}
}
