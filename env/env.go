package env

import (
	"os"
	"strconv"
)

type Env struct {
	Server  ServerConfig
	MongoDB MongoDBConfig
}
type ServerConfig struct {
	Port int
}

type MongoDBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DB       string
}

var setupEnv = false
var env = Env{}

func GetEnv() (*Env, error) {

	if !setupEnv {

		//TODO: Find better way since flag does not parse properly in test
		mongoDBHost := os.Getenv("MONGODB_HOST")
		mongoDBPort := os.Getenv("MONGODB_PORT")
		parsedMongoDBPort, err := strconv.Atoi(mongoDBPort)
		if err != nil {
			return nil, err
		}

		mongoDBUser := os.Getenv("MONGODB_USER")
		mongoDBPassword := os.Getenv("MONGODB_PASSWORD")
		mongoDBName := os.Getenv("MONGODB_NAME")

		serverPort := os.Getenv("SERVER_PORT")
		parsedServerPort, err := strconv.Atoi(serverPort)
		if err != nil {
			return nil, err
		}

		env = Env{
			Server: ServerConfig{
				Port: parsedServerPort,
			},
			MongoDB: MongoDBConfig{
				Host:     mongoDBHost,
				Port:     parsedMongoDBPort,
				User:     mongoDBUser,
				Password: mongoDBPassword,
				DB:       mongoDBName,
			},
		}
	}

	return &env, nil
}
