package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// GetLeaderIP returns the IP of the leader of the current bootstrap nodes
func GetLeaderIP(listOfBootstrapNodes []string) (string, error) {
	log.Println(listOfBootstrapNodes)
	for _, ip := range listOfBootstrapNodes {
		log.Println("Trying to check status of " + ip)
		url := fmt.Sprintf("http://%s:%d/status", ip, DBApiPort)
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
