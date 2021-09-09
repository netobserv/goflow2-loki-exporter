package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/backoff"
	"github.com/grafana/loki-client-go/pkg/urlutil"
	promconf "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	URL                    string                    `yaml:"url"`
	TenantID               string                    `yaml:"tenantID"`
	BatchWaitSeconds       int64                     `yaml:"batchWaitSeconds"`
	BatchSize              int                       `yaml:"batchSize"`
	TimeoutSeconds         int64                     `yaml:"timeoutSeconds"`
	MinBackoffSeconds      int64                     `yaml:"minBackoffSeconds"`
	MaxBackoffSeconds      int64                     `yaml:"maxBackoffSeconds"`
	MaxRetries             int                       `yaml:"maxRetries"`
	Labels                 []model.LabelName         `yaml:"labels"`
	StaticLabels           model.LabelSet            `yaml:"staticLabels"`
	IgnoreList             []model.LabelName         `yaml:"ignoreList"`
	PrintInput             bool                      `yaml:"printInput"`
	PrintOutput            bool                      `yaml:"printOutput"`
	ClientConfig           promconf.HTTPClientConfig `yaml:"clientConfig"`
	TimestampLabel         model.LabelName           `yaml:"timestampLabel"`
	TimestampScaleToSecond float64                   `yaml:"timestampScaleToSecond"`
}

func Load(file string) (Config, error) {
	c := Default()
	log.Tracef("Reading YAML config from %s", file)
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return c, fmt.Errorf("Failed to load config: %v", err)
	}
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return c, fmt.Errorf("Failed to parse YAML: %v", err)
	}
	log.Tracef("Config: %v", c)

	return c, nil
}

func Default() Config {
	return Config{
		URL:               "http://loki:3100/",
		BatchWaitSeconds:  1,
		BatchSize:         100 * 1024,
		TimeoutSeconds:    10,
		MinBackoffSeconds: 1,
		MaxBackoffSeconds: 5 * 60,
		MaxRetries:        10,
		StaticLabels: model.LabelSet{
			"app": "goflow2",
		},
		TimestampLabel:         "TimeFlowStart",
		TimestampScaleToSecond: 1,
	}
}

func (c *Config) BuildClientConfig() (loki.Config, error) {
	cfg := loki.Config{
		TenantID:  c.TenantID,
		BatchWait: time.Second * time.Duration(c.BatchWaitSeconds),
		BatchSize: int(c.BatchSize),
		Timeout:   time.Second * time.Duration(c.TimeoutSeconds),
		BackoffConfig: backoff.BackoffConfig{
			MinBackoff: time.Second * time.Duration(c.MinBackoffSeconds),
			MaxBackoff: time.Second * time.Duration(c.MaxBackoffSeconds),
			MaxRetries: c.MaxRetries,
		},
		Client: c.ClientConfig,
	}
	var clientURL urlutil.URLValue
	err := clientURL.Set(strings.TrimSuffix(c.URL, "/") + "/loki/api/v1/push")
	if err != nil {
		return cfg, errors.New("failed to parse client URL")
	}
	cfg.URL = clientURL
	return cfg, nil
}
