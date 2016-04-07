package web

import (
	"io/ioutil"
	"log"
	"net/http"
    “fmt”
	"github.com/flosch/pongo2"
)

var tplIndex = pongo2.Must(pongo2.FromFile("web/templates/index.html"))
var tplDraw = pongo2.Must(pongo2.FromFile("web/templates/draw.html"))
var tplLogin = pongo2.Must(pongo2.FromFile("web/templates/login.html"))
var tplRoom = pongo2.Must(pongo2.FromFile("web/templates/gameRoom.html"))


// httpError returns a HTTP 5xx error
func httpError(err error, w http.ResponseWriter) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// RoomList returns a page with the list of rooms that are available to join
func gameRoom(w http.ResponseWriter, r *http.Request) {
	//url := Config.SupernodeURL + "/gameRoom"
    url := "http://localhost:8000/gameRoom/" //abselote localhost path not working

	res, err := http.Get(url)
	httpError(err, w)
	defer res.Body.Close()
	contents, err := ioutil.ReadAll(res.Body)
	httpError(err, w)
    //?????what does this part do?????
	err = tplRoom.ExecuteWriter(pongo2.Context{"testValue": string(contents)}, w)
	httpError(err, w)
}

// Login page handler
func login(w http.ResponseWriter, r *http.Request) {
    //fmt.Fprintln(w, "Welcome!")
	//url := Config.SupernodeURL + "login"
    url := "http://localhost:8000/login.html" //not working
    //url := "http://localhost:8000/login.html"  not working

    //fmt.Printf("inside login handler, with url:" + url);
	res, err := http.Get(url)
    defer res.Body.Close()
	contents, err := ioutil.ReadAll(res.Body)
	httpError(err, w)
	err = tplLogin.ExecuteWriter(pongo2.Context{"testValue": "tests", w)
	httpError(err, w)
	
}

//Index handler handles the landing page of the UI
func Index(w http.ResponseWriter, r *http.Request) {
	err := tplIndex.ExecuteWriter(pongo2.Context{"testValue": "Hello World"}, w)
	httpError(err, w)
}





//Index handler handles the landing page of the UI
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
