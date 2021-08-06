package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/grafana/loki/pkg/promtail/client"
	promconf "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	URL               string
	TenantID          string
	BatchWaitSeconds  int64
	BatchSize         int
	TimeoutSeconds    int64
	MinBackoffSeconds int64
	MaxBackoffSeconds int64
	MaxRetries        int
	Labels            []model.LabelName
	StaticLabels      model.LabelSet
	IgnoreList        []model.LabelName
	PrintRecords      bool
	ClientConfig      promconf.HTTPClientConfig
}

func Load(file string) (Config, error) {
	c := Default()
	log.Debugf("Reading YAML config from %s", file)
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return c, fmt.Errorf("Failed to load config: %v", err)
	}
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return c, fmt.Errorf("Failed to parse YAML: %v", err)
	}

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
	}
}

func (c *Config) BuildClientConfig() (client.Config, error) {
	cfg := client.Config{
		TenantID:  c.TenantID,
		BatchWait: time.Second * time.Duration(c.BatchWaitSeconds),
		BatchSize: int(c.BatchSize),
		Timeout:   time.Second * time.Duration(c.TimeoutSeconds),
		BackoffConfig: util.BackoffConfig{
			MinBackoff: time.Second * time.Duration(c.MinBackoffSeconds),
			MaxBackoff: time.Second * time.Duration(c.MaxBackoffSeconds),
			MaxRetries: c.MaxRetries,
		},
		Client: c.ClientConfig,
	}
	var clientURL flagext.URLValue
	err := clientURL.Set(strings.TrimSuffix(c.URL, "/") + "/loki/api/v1/push")
	if err != nil {
		return cfg, errors.New("failed to parse client URL")
	}
	cfg.URL = clientURL
	return cfg, nil
}
