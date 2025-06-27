package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	Environment   string   `yaml:"environment,omitempty"`
	Customer      string   `yaml:"customer,omitempty"`
	VpcCidr       string   `yaml:"vpc_cidr,omitempty"`
	Azs           []string `yaml:"availability_zones,omitempty"`
	eksVersion    string   `yaml:"eks_version,omitempty"`
	ProductDomain string   `yaml:"product_domain,omitempty"`
}

func LoadConfig(configPath string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

// GetCustomerName returns the customer name regardless of config format
func (c *Config) GetCustomerName() string {
	return c.Customer
}
