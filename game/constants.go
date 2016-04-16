package game

const (
	//ConfigFileName holds the name of the config file
	ConfigFileName string = "game/config.json"
	// DBFolder holds the path to the database folder
	DBFolder string = "./roomdb"
	// GameDBApiPort is the port on which the DB's API is hosted
	GameDBApiPort int = 4003
	// GameDBRaftPort is the port on which the raft comm takes place
	GameDBRaftPort int = 4004
)
