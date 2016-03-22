package web

import (
	"io/ioutil"
	"net/http"

	"github.com/flosch/pongo2"
)

var tplIndex = pongo2.Must(pongo2.FromFile("web/templates/index.html"))
var tplDraw = pongo2.Must(pongo2.FromFile("web/templates/draw.html"))

func perror(err error, w http.ResponseWriter) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//Index handler handles the landing page of the UI
func Index(w http.ResponseWriter, r *http.Request) {
	err := tplIndex.ExecuteWriter(pongo2.Context{"testValue": "Hello World"}, w)
	perror(err, w)
}

func RoomList(w http.ResponseWriter, r *http.Request) {
	url := Config.SupernodeURL + "/GetPeersForRoom/default"
	res, err := http.Get(url)
	perror(err, w)
	defer res.Body.Close()
	contents, err := ioutil.ReadAll(res.Body)
	perror(err, w)
	err = tplIndex.ExecuteWriter(pongo2.Context{"testValue": string(contents)}, w)
	perror(err, w)
}

//Index handler handles the landing page of the UI
func Draw(w http.ResponseWriter, r *http.Request) {
	err := tplIndex.ExecuteWriter(pongo2.Context{"testValue": "Hello World"}, w)
	perror(err, w)
}
