package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Backend struct {
	Id     int    `json:"id"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Weight int    `json:"weight"`
}

type Frontend struct {
	BindPort int       `json:"bindPort"`
	Backends []Backend `json:"backends"`
}

type Config struct {
	Frontends []Frontend `json:"frontends"`
}

var singleConfig Config

func init() {

	configFilePath, exists := os.LookupEnv("BRELAY_CONFIG_FILE")
	if !exists {
		configFilePath = "/etc/brelay.conf"
	}

	data, err := ioutil.ReadFile(configFilePath)

	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &singleConfig)

	if err != nil {
		panic(err)
	}

}

func Get() Config {
	return singleConfig
}
