package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/flosch/pongo2"
)

var tplIndex = pongo2.Must(pongo2.FromFile("web/templates/index.html"))
var tplLogin = pongo2.Must(pongo2.FromFile("web/templates/login.html"))
var tplRooms = pongo2.Must(pongo2.FromFile("web/templates/rooms.html"))
var tplDraw = pongo2.Must(pongo2.FromFile("web/templates/draw.html"))

// httpError returns a HTTP 5xx error
func httpError(err error, w http.ResponseWriter) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//Index handler handles the landing page of the UI
func Index(w http.ResponseWriter, r *http.Request) {
	err := tplIndex.ExecuteWriter(pongo2.Context{"testValue": "Hello World"}, w)
	httpError(err, w)
}

//Login handler handles the login page of the UI
func Login(w http.ResponseWriter, r *http.Request) {
	ip, _ := GetPublicIP()
	err := tplLogin.ExecuteWriter(pongo2.Context{"dns": Config.BootstrapDNSEndpoint, "ip": ip, "nickname": Nickname}, w)
	httpError(err, w)
}

// AuthUser is used to set the current user
func AuthUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.ParseForm()

	name := r.FormValue("nickname")
	if name == "" {
		http.Error(w, "nickname not loggedin", http.StatusInternalServerError)
		return
	}
	Nickname = name
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

// Logout is used to set the current user
func Logout(w http.ResponseWriter, r *http.Request) {
	url := Config.BootstrapDNSEndpoint + "/player/delete/" + Nickname
	log.Println("Delete url: " + url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "nickname not logged out", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("nickname not logged out: %s", resp.Status)
		http.Error(w, "nickname not logged out", http.StatusInternalServerError)
		return
	}
	Nickname = ""
	Login(w, r)
}

// RoomList returns a page with the list of rooms that are available to join
func RoomList(w http.ResponseWriter, r *http.Request) {
	// url := Config.BootstrapDNSEndpoint + "/rooms/"
	// res, err := http.Get(url)
	// httpError(err, w)
	// defer res.Body.Close()
	// contents, err := ioutil.ReadAll(res.Body)
	// httpError(err, w)

	if Nickname == "" {
		Login(w, r)
		return
	}
	err := tplRooms.ExecuteWriter(pongo2.Context{"nickname": Nickname, "dns": Config.BootstrapDNSEndpoint}, w)
	httpError(err, w)
}

// Game handler handles the landing page of the UI
func Game(w http.ResponseWriter, r *http.Request) {
	err := tplIndex.ExecuteWriter(pongo2.Context{"testValue": "Hello World"}, w)
	httpError(err, w)
}

// HandleSocketConn is used as the endpoint fot websocket connections to be made
func HandleSocketConn(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	c := &connection{send: make(chan []byte, 256), ws: ws}
	Hub.register <- c
	go c.WriteMessagesToSocket()
	c.ReadMessagesFromSocket()
}
