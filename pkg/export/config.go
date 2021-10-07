// Package export provides configuration management functions
package export

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/backoff"
	"github.com/grafana/loki-client-go/pkg/urlutil"
	promconf "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
)

var clog = logrus.WithField("module", "export/config")

type Config struct {
	URL            string                    `yaml:"url"`
	TenantID       string                    `yaml:"tenantID"`
	BatchWait      time.Duration             `yaml:"batchWait"`
	BatchSize      int                       `yaml:"batchSize"`
	Timeout        time.Duration             `yaml:"timeout"`
	MinBackoff     time.Duration             `yaml:"minBackoff"`
	MaxBackoff     time.Duration             `yaml:"maxBackoff"`
	MaxRetries     int                       `yaml:"maxRetries"`
	Labels         []string                  `yaml:"labels"`
	StaticLabels   model.LabelSet            `yaml:"staticLabels"`
	IgnoreList     []string                  `yaml:"ignoreList"`
	PrintInput     bool                      `yaml:"printInput"`
	PrintOutput    bool                      `yaml:"printOutput"`
	ClientConfig   promconf.HTTPClientConfig `yaml:"clientConfig"`
	TimestampLabel model.LabelName           `yaml:"timestampLabel"`
	// TimestampScale provides the scale in time of the units from the timestamp
	// E.g. UNIX time scale is '1s' (one second) while other clock sources might have
	// scales of '1ms' (one millisecond) or just '1' (one nanosecond)
	// Default value is '1s'
	TimestampScale time.Duration `yaml:"timestampScale"`
}

// LoadConfig loads the YAML configuration from the file path passed as argument
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ReadConfig(file)
}

// ReadConfig reads a YAML configuration from the io.Reader passed as argument
func ReadConfig(in io.Reader) (*Config, error) {
	bytes, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	c := DefaultConfig()
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	clog.WithField("config", c).Debug("loaded configuration")
	return c, nil
}

func DefaultConfig() *Config {
	return &Config{
		URL:        "http://loki:3100/",
		BatchWait:  1,
		BatchSize:  100 * 1024,
		Timeout:    10,
		MinBackoff: 1,
		MaxBackoff: 5 * 60,
		MaxRetries: 10,
		StaticLabels: model.LabelSet{
			"app": "goflow2",
		},
		TimestampLabel: "TimeReceived",
		TimestampScale: time.Second,
	}
}

func (c *Config) buildLokiConfig() (loki.Config, error) {
	cfg := loki.Config{
		TenantID:  c.TenantID,
		BatchWait: c.BatchWait,
		BatchSize: c.BatchSize,
		Timeout:   c.Timeout,
		BackoffConfig: backoff.BackoffConfig{
			MinBackoff: c.MinBackoff,
			MaxBackoff: c.MaxBackoff,
			MaxRetries: c.MaxRetries,
		},
		Client: c.ClientConfig,
	}
	var clientURL urlutil.URLValue
	err := clientURL.Set(strings.TrimSuffix(c.URL, "/") + "/loki/api/v1/push")
	if err != nil {
		return cfg, fmt.Errorf("failed to parse client URL: %w", err)
	}
	cfg.URL = clientURL
	return cfg, nil
}

func validate(c *Config) error {
	if c == nil {
		return errors.New("you must provide a configuration")
	}
	if c.TimestampScale == 0 {
		return errors.New("timestampUnit must be a valid Duration > 0 (e.g. 1m, 1s or 1ms)")
	}
	if c.URL == "" {
		return errors.New("url can't be empty")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("invalid batchSize: %v. Required > 0", c.BatchSize)
	}
	return nil
}
