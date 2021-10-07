// Package export enables data exporting to data ingestion backends (e.g. Loki)
package export

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	logadapter "github.com/go-kit/kit/log/logrus"

	"github.com/grafana/loki-client-go/loki"
	jsoniter "github.com/json-iterator/go"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

var (
	keyReplacer = strings.NewReplacer("/", "_", ".", "_", "-", "_")
	log         = logrus.WithField("module", "export/loki")
)

// Emitter abstracts the records' ingester (e.g. the Loki client)
type emitter interface {
	Handle(labels model.LabelSet, timestamp time.Time, record string) error
}

// Loki record exporter
type Loki struct {
	config     Config
	lokiConfig loki.Config
	emitter    emitter
	timeNow    func() time.Time
}

// NewLoki creates a Loki flow exporter from a given configuration
func NewLoki(cfg *Config) (Loki, error) {
	if err := validate(cfg); err != nil {
		return Loki{}, fmt.Errorf("the provided config is not valid: %w", err)
	}
	lcfg, err := cfg.buildLokiConfig()
	if err != nil {
		return Loki{}, err
	}
	lokiClient, err := loki.NewWithLogger(lcfg, logadapter.NewLogrusLogger(log))
	if err != nil {
		return Loki{}, err
	}
	return Loki{
		config:     *cfg,
		lokiConfig: lcfg,
		emitter:    lokiClient,
		timeNow:    time.Now,
	}, nil
}

// Process the flows provided as JSON lines by the input io.Reader until the end of the file
func (l *Loki) Process(in io.Reader) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Bytes()
		if l.config.PrintInput {
			fmt.Println(string(line))
		}
		err := l.processRecord(line)
		if err != nil {
			log.Error(err)
		}
	}
	return scanner.Err()
}

func (l *Loki) processRecord(rawRecord []byte) error {
	// TODO: allow protobuf input
	var record map[string]interface{}
	err := json.Unmarshal(rawRecord, &record)
	if err != nil {
		return err
	}

	// Get timestamp from record (default: TimeFlowStart)
	timestamp := l.extractTimestamp(record)

	labels := model.LabelSet{}

	// Add static labels from config
	for k, v := range l.config.StaticLabels {
		labels[k] = v
	}

	l.addNonStaticLabels(record, labels)

	// Remove labels and configured ignore list from record
	ignoreList := append(l.config.IgnoreList, l.config.Labels...)
	for _, label := range ignoreList {
		delete(record, label)
	}

	js, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(record)
	if err != nil {
		return err
	}
	if l.config.PrintOutput {
		fmt.Println(string(js))
	}
	return l.emitter.Handle(labels, timestamp, string(js))
}

func (l *Loki) extractTimestamp(record map[string]interface{}) time.Time {
	if l.config.TimestampLabel == "" {
		return l.timeNow()
	}
	timestamp, ok := record[string(l.config.TimestampLabel)]
	if !ok {
		log.WithField("timestampLabel", l.config.TimestampLabel).
			Warnf("Timestamp label not found in record. Using local time")
		return l.timeNow()
	}
	ft, ok := timestamp.(float64)
	if !ok {
		log.WithField(string(l.config.TimestampLabel), timestamp).
			Warnf("Invalid timestamp found: number expected. Using local time")
		return l.timeNow()
	}
	if ft == 0 {
		log.WithField("timestampLabel", l.config.TimestampLabel).
			Warnf("Empty timestamp in record. Using local time")
		return l.timeNow()
	}
	tsNanos := int64(ft * float64(l.config.TimestampScale))
	return time.Unix(tsNanos/int64(time.Second), tsNanos%int64(time.Second))
}

func (l *Loki) addNonStaticLabels(record map[string]interface{}, labels model.LabelSet) {
	// Add non-static labels from record
	for _, label := range l.config.Labels {
		val, ok := record[label]
		if !ok {
			continue
		}
		sanitizedKey := model.LabelName(keyReplacer.Replace(label))
		if !sanitizedKey.IsValid() {
			log.WithFields(logrus.Fields{"key": label, "sanitizedKey": sanitizedKey}).
				Debug("Invalid label. Ignoring it")
			continue
		}
		lv := model.LabelValue(fmt.Sprint(val))
		if !lv.IsValid() {
			log.WithFields(logrus.Fields{"key": label, "sanitizedKey": sanitizedKey, "value": val}).
				Debug("Invalid label value. Ignoring it")
			continue
		}
		labels[sanitizedKey] = lv
	}
}
