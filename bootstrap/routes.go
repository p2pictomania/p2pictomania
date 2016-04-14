package bootstrap

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Route struct holds the handler, name and pattern for each route
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes holds the list of routs supported by bootstrap
type Routes []Route

// NewRouter returns a new Gorilla mux router based on the configured routes
func NewRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(route.HandlerFunc)
	}
	return router
}

var routes = Routes{
	Route{
		"Index",
		"GET",
		"/",
		Index,
	},
	Route{
		"GetPeers",
		"GET",
		"/peers/{roomid}",
		GetPeersForRoom,
	},
	Route{
		"PlayerJoin",
		"POST",
		"/player/join",
		AddPlayerToRoom,
	},
	Route{
		"PlayerNew",
		"POST",
		"/player/new",
		AddNewPlayer,
	},
	Route{
		"PlayerDelete",
		"GET",
		"/player/delete/{nickname}",
		DeletePlayer,
	},
	Route{
		"PlayerLeave",
		"POST",
		"/player/leave",
		DeletePlayerFromRoom,
	},
	Route{
		"RoomNew",
		"POST",
		"/room/new",
		CreateNewRoom,
	},
	Route{
		"RoomOpen",
		"GET",
		"/room/open/{roomID}",
		OpenRoom,
	},
	Route{
		"RoomList",
		"GET",
		"/rooms",
		GetRoomsList,
	},
}
