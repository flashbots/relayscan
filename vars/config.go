package vars

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration loaded from YAML
type Config struct {
	Genesis          int64               `yaml:"genesis"`
	Relays           RelaysConfig        `yaml:"relays"`
	BuilderAddresses map[string][]string `yaml:"builder_addresses"`
}

// RelaysConfig holds relay URL configuration
type RelaysConfig struct {
	Flashbots  string   `yaml:"flashbots"`
	Ultrasound string   `yaml:"ultrasound"`
	All        []string `yaml:"all"`
}

var loadedConfig *Config

// LoadConfig loads the configuration from a YAML file
func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	loadedConfig = &cfg

	// Populate package-level variables for backwards compatibility
	Genesis = int(cfg.Genesis)
	RelayFlashbots = cfg.Relays.Flashbots
	RelayUltrasound = cfg.Relays.Ultrasound
	RelayURLs = cfg.Relays.All
	BuilderAddresses = buildAddressMap(cfg.BuilderAddresses)

	return nil
}

// MustLoadConfig loads the configuration or panics on error
func MustLoadConfig(path string) {
	if err := LoadConfig(path); err != nil {
		panic(err)
	}
}

// GetConfig returns the loaded configuration
func GetConfig() *Config {
	return loadedConfig
}

// buildAddressMap converts the config format to the expected map[coinbase]map[address]bool format
func buildAddressMap(addresses map[string][]string) map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	for coinbase, addrs := range addresses {
		result[coinbase] = make(map[string]bool)
		for _, addr := range addrs {
			result[coinbase][addr] = true
		}
	}
	return result
}
