package bootstrap

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

//Index handler handles the landing page of the UI
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi")
}

// GetPeersForRoom handler retuns the peers for a given room name
func GetPeersForRoom(w http.ResponseWriter, r *http.Request) {
	urlVars := mux.Vars(r)
	log.Printf("Peers for room %s requested", urlVars["roomname"])

	payload := PeerList{[]string{"127.0.0.1"}}

	json.NewEncoder(w).Encode(payload)
}
