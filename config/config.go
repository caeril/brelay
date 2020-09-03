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

type Path struct {
	Path     string    `json:"path"`
	Backends []Backend `json:"backends"`
}

type Host struct {
	Hostname string `json:"hostname"`
	Paths    []Path `json:"paths"`
}

type Frontend struct {
	BindPort    int    `json:"bind_port"`
	TLSKeyPath  string `json:"tls_key_path"`
	TLSCertPath string `json:"tls_cert_path"`
	Hosts       []Host `json:"hosts"`
}

type Logging struct {
	AccessPath string `json:"access_path"`
	ErrorPath  string `json:"error_path"`
}

type Config struct {
	Logging   Logging    `json:"logging"`
	Frontends []Frontend `json:"frontends"`
}

var singleConfig Config

func init() {

	configFilePath, exists := os.LookupEnv("BRELAY_CONFIG_FILE")
	if !exists {
		configFilePath = "/etc/brelay/brelay.conf"
	}

	data, err := ioutil.ReadFile(configFilePath)

	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &singleConfig)

	if err != nil {
		panic(err)
	}

	if len(singleConfig.Logging.AccessPath) < 2 {
		singleConfig.Logging.AccessPath = "/var/log/brelay/access.log"
	}

	if len(singleConfig.Logging.ErrorPath) < 2 {
		singleConfig.Logging.ErrorPath = "/var/log/brelay/error.log"
	}

}

func Get() Config {
	return singleConfig
}
