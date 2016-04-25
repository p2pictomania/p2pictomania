package web

const (
	//ConfigFileName holds the name of the config file
	ConfigFileName string = "web/config.json"
	//MaxRoomPlayers hols the max number of players in a room
	MaxRoomPlayers int = 2
	// RoomTimeLimit is the time limit of the room
	RoomTimeLimit int = 60000
	// GameDBFolder holds the path to the database folder
	GameDBFolder string = "./gamedb"
	// GameDBApiPort is the port on which the DB's API is hosted
	GameDBApiPort int = 4003
	// GameDBRaftPort is the port on which the raft comm takes place
	GameDBRaftPort int = 4004
)
