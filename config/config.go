package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	guerrilla "github.com/flashmob/go-guerrilla"
)

// ReadConfig which should be called at startup, or when a SIG_HUP is caught
func ReadConfig(configFile, iface string, verbose bool, mainConfig *guerrilla.Config) error {
	// load in the config.
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("could not read config file: %s", err)
	}

	err = json.Unmarshal(b, &mainConfig)
	if err != nil {
		return fmt.Errorf("could not parse config file: %s", err)
	}

	// TODO: deprecate
	if len(iface) > 0 && len(mainConfig.Servers) > 0 {
		mainConfig.Servers[0].Listen_interface = iface
	}
	// TODO: deprecate
	if verbose {
		mainConfig.Verbose = true
	}

	guerrilla.ConfigLoadTime = time.Now()
	return nil
}
