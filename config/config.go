package config

type Backend struct {
	Id     int
	Host   string
	Port   int
	Weight int
}

type Config struct {
	BindPort int
	Backends []Backend
}

var singleConfig Config

func init() {
	singleConfig = Config{BindPort: 2021}
	singleConfig.Backends = append(singleConfig.Backends, Backend{Id: 1, Host: "127.0.0.1", Port: 8001})
}

func Get() Config {
	return singleConfig
}
