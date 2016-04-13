package game

const (
	//ConfigFileName holds the name of the config file
	ConfigFileName string = "game/config.json"
	// DBFolder holds the path to the database folder
	DBFolder string = "./roomdb"
	// DBApiPort is the port on which the DB's API is hosted
	DBApiPort int = 9001
	// DBRaftPort is the port on which the raft comm takes place
	DBRaftPort int = 9002
)
