package config

import (
	"log"
	"sync"

	env "github.com/caarlos0/env/v7"
)

//docker run --name node1 -p 9042:9042 -d scylladb/scylla --broadcast-address 127.0.0.1 --listen-address 0.0.0.0 --broadcast-rpc-address 127.0.0.1

var config *Config
var once sync.Once

type Config struct {
	Auth struct {
		PhoneNum string `env:"APP_PHN,file" envDefault:"C:/Users/kmr3x/OneDrive/Desktop/WnEd/Go_Projects/CFG_INFO/APP_PHN.txt"`
		AppId    string `env:"APPID_ISUB,file" envDefault:"C:/Users/kmr3x/OneDrive/Desktop/WnEd/Go_Projects/CFG_INFO/APPID_ISUB.txt"`
		AppHash  string `env:"APPHASH_ISUB,file" envDefault:"C:/Users/kmr3x/OneDrive/Desktop/WnEd/Go_Projects/CFG_INFO/APPHASH_ISUB.txt"`
	}
}

func Init() {
	once.Do(func() {
		log.SetFlags(0)
		config = &Config{}
		err := env.Parse(config)
		if err != nil {
			log.Fatalf("failed to parse config: %v", err)
		}
	})
}

func Get() *Config {
	return config
}
