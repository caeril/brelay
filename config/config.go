package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Backend struct {
	Id       int    `json:"id"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Weight   int    `json:"weight"`
}

type Frontend struct {
	BindPort    int       `json:"bind_port"`
	Hostname    string    `json:"hostname"`
	Backends    []Backend `json:"backends"`
	TLSKeyPath  string    `json:"tls_key_path"`
	TLSCertPath string    `json:"tls_cert_path"`
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
