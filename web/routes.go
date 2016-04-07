package web

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

func NewRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(route.HandlerFunc).GetError()
	}

	//Setting up routes for static files
	s := http.StripPrefix(Config.StaticUrlPrefix, http.FileServer(http.Dir(Config.StaticDir)))
	router.PathPrefix(Config.StaticUrlPrefix).Handler(s)
//router.PathPrefix("/").Handler(s) 
	return router
}

var routes = Routes{
	Route{
		"WebSock",
		"GET",
		"/ws/",
		HandleSocketConn,
	},
	Route{
		"gameRoom",
		"GET",
		"/gameRoom/",
		gameRoom,
	},
    Route{
		"login",
		"GET",
		"/login/",
		login,
	},
	Route{
		"Draw",
		"GET",
		"/Draw/",
		Draw,
	},
    Route{
		"Index",
		"GET",
		"/",
		Index,
	},
}
