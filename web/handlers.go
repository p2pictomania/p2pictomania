package web

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/flosch/pongo2"
)

var tplIndex = pongo2.Must(pongo2.FromFile("web/templates/index.html"))
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

// RoomList returns a page with the list of rooms that are available to join
func RoomList(w http.ResponseWriter, r *http.Request) {
	url := Config.BootstrapNodeURL + "/peers/default"
	res, err := http.Get(url)
	httpError(err, w)
	defer res.Body.Close()
	contents, err := ioutil.ReadAll(res.Body)
	httpError(err, w)
	err = tplIndex.ExecuteWriter(pongo2.Context{"testValue": string(contents)}, w)
	httpError(err, w)
}

// Draw handler handles the landing page of the UI
func Draw(w http.ResponseWriter, r *http.Request) {
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
