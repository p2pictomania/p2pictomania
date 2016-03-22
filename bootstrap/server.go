package bootstrap

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

var Config ConfigObject

//ConfigObject holds the parsed config.json file
type ConfigObject struct {
	Port int `json:"port"`
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

//StartServer is the function used to start the local server used to
//interact with the web interface
func StartServer() {
	Config = parseConfigFile()
	router := NewRouter()
	log.Println("Starting Supernode server on port " + strconv.Itoa(Config.Port))
	err := http.ListenAndServe(":"+strconv.Itoa(Config.Port), router)
	if err != nil {
		log.Fatalf("Failed to start the UI server: %s", err)
	}
}
