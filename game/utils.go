package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// GetLeaderIP returns the IP of the leader of the current bootstrap nodes
func GetLeaderIP(listOfRoomNodes []string) (string, error) {
	for _, ip := range listOfRoomNodes {
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

// GetListOfPlayersForRoom does shit
func GetListOfPlayersForRoom(roomID string) ([]interface{}, error) {
	url := "http://autogra.de/peers/" + roomID
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
