package web

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

// Routes holds the list of routs supported by web
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

	//Setting up routes for static files
	s := http.StripPrefix(Config.StaticURLPrefix, http.FileServer(http.Dir(Config.StaticDir)))
	router.PathPrefix(Config.StaticURLPrefix).Handler(s)

	return router
}

var routes = Routes{
	Route{
		"Index",
		"GET",
		"/",
		Login,
	},
	Route{
		"Auth",
		"POST",
		"/auth",
		AuthUser,
	},
	Route{
		"Logout",
		"GET",
		"/logout",
		Logout,
	},
	Route{
		"RoomList",
		"GET",
		"/rooms",
		RoomList,
	},
	Route{
		"Draw",
		"GET",
		"/game/{roomID}",
		Game,
	},
	Route{
		"WebSock",
		"GET",
		"/ws",
		HandleSocketConn,
	},
}
