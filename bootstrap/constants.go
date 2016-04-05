package bootstrap

const (
	// BootstrapPort is where the bootstrap server will bind on
	BootstrapPort int = 5000
	// MaxNumBootstrapNode is the max number of bootstrap nodes in the network
	MaxNumBootstrapNode int = 3
	//ConfigFileName holds the name of the config file
	ConfigFileName string = "bootstrap/config.json"
	// DBFolder holds the path to the database folder
	DBFolder string = "./db"
	// DBApiPort is the port on which the DB's API is hosted
	DBApiPort int = 4001
	// DBRaftPort is the port on which the raft comm takes place
	DBRaftPort int = 4002
)
