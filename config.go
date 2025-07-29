package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/chpc-uofu/cgroup-warden/hierarchy"
	"github.com/containerd/cgroups/v3/cgroup2"
)

type Config struct {
	RootCGroup    string  `env:"ROOT_CGROUP" envDefault:"/user.slice"`
	ListenAddress string  `env:"LISTEN_ADDRESS" envDefault:":2112"`
	Certificate   string  `env:"CERTIFICATE"`
	PrivateKey    string  `env:"PRIVATE_KEY"`
	BearerToken   string  `env:"BEARER_TOKEN"`
	InsecureMode  bool    `env:"INSECURE_MODE" envDefault:"false"`
	MetaMetrics   bool    `env:"META_METRICS" envDefault:"true"`
	LogLevel      string  `env:"LOG_LEVEL" envDefault:"info"`
	SwapRatio     float64 `env:"SWAP_RATIO" envDefault:"0.1"`
}

func NewConfig() (*Config, error) {
	var c Config
	var err error

	err = env.ParseWithOptions(&c, env.Options{Prefix: "CGROUP_WARDEN_"})
	if err != nil {
		return nil, err
	}

	err = cgroup2.VerifyGroupPath(c.RootCGroup)
	if err != nil {
		return nil, fmt.Errorf("Invalid cgroup root: '%v'", c.RootCGroup)
	}

	if !c.InsecureMode {

		if c.Certificate == "" {
			return nil, fmt.Errorf("Certificate required if not running in insecure mode")
		}

		if c.PrivateKey == "" {
			return nil, fmt.Errorf("Private key required if not running insecure mode")
		}

		if c.BearerToken == "" {
			return nil, fmt.Errorf("Bearer token required if not running in insecure mode")
		}
	}

	levels := []string{"info", "warning", "debug", "error"}
	c.LogLevel = strings.ToLower(c.LogLevel)

	if !slices.Contains(levels, c.LogLevel) {
		return nil, fmt.Errorf("Invalid log level. Options include %v", levels)
	}

	hierarchy.SwapRatio = c.SwapRatio

	return &c, err
}
