package guerrilla

type BackendConfig map[string]interface{}

// Config is the holder of the configuration of the app
type Config struct {
	BackendName   string         `json:"backend_name"`
	BackendConfig BackendConfig  `json:"backend_config,omitempty"`
	Servers       []ServerConfig `json:"servers"`
	Verbose       bool           `json:"verbose"`

	Allowed_hosts string `json:"allowed_hosts"`
}

// ServerConfig is the holder of the configuration of a server
type ServerConfig struct {
	IsEnabled        bool   `json:"is_enabled"`
	Host_name        string `json:"host_name"`
	Max_size         int    `json:"max_size"`
	Private_key_file string `json:"private_key_file"`
	Public_key_file  string `json:"public_key_file"`
	Timeout          int    `json:"timeout"`
	Listen_interface string `json:"listen_interface"`
	Start_tls_on     bool   `json:"start_tls_on,omitempty"`
	Tls_always_on    bool   `json:"tls_always_on,omitempty"`
	Max_clients      int    `json:"max_clients"`
	Log_file         string `json:"log_file"`
}
