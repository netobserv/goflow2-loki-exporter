package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	logadapter "github.com/go-kit/kit/log/logrus"
	"github.com/grafana/loki-client-go/loki"
	"github.com/jotak/goflow2-loki-exporter/config"
	jsoniter "github.com/json-iterator/go"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

var (
	version     = "unknown"
	app         = "loki-exporter"
	configFile  = flag.String("config", "", "Path to the YAML config file")
	logLevel    = flag.String("loglevel", "info", "Log level")
	versionFlag = flag.Bool("v", false, "Print version")
	log         = logrus.WithField("module", app)
	appVersion  = fmt.Sprintf("%s %s", app, version)
)

var (
	keyReplacer = strings.NewReplacer("/", "_", ".", "_", "-", "_")
)

func init() {
}

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(appVersion)
		os.Exit(0)
	}

	lvl, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("Log level %s not recognized, using info", *logLevel)
		*logLevel = "info"
		lvl = logrus.InfoLevel
	}
	logrus.SetLevel(lvl)

	var conf config.Config
	if *configFile != "" {
		conf, err = config.Load(*configFile)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Infof("Config file not provided, using defaults")
		conf = config.Default()
	}

	log.Infof("Starting %s at log level %s", appVersion, *logLevel)

	clientConfig, err := conf.BuildClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	lokiClient, err := loki.NewWithLogger(clientConfig, logadapter.NewLogrusLogger(log))
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		in := scanner.Bytes()
		if conf.PrintInput {
			fmt.Println(string(in))
		}
		err := processRecord(in, conf, lokiClient)
		if err != nil {
			log.Error(err)
		}
	}

	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func processRecord(rawRecord []byte, conf config.Config, lokiClient *loki.Client) error {
	// TODO: allow protobuf input
	var record map[model.LabelName]interface{}
	err := json.Unmarshal(rawRecord, &record)
	if err != nil {
		return err
	}

	// Get timestamp from record (default: TimeFlowStart)
	timestamp := time.Now()
	if conf.TimestampLabel != "" {
		if t, ok := record[conf.TimestampLabel]; ok {
			if ft, ok := t.(float64); ok {
				unix := int64(ft * conf.TimestampScaleToSecond)
				if unix != 0 {
					timestamp = time.Unix(unix, 0)
				} else {
					log.Warnf("Empty timestamp (%s) found in record, use now instead", conf.TimestampLabel)
				}
			} else {
				log.Warnf("Invalid timestamp (%s: %v) found in record: number explected", conf.TimestampLabel, t)
			}
		} else {
			log.Warnf("Timestamp label %s not found in record", conf.TimestampLabel)
		}
	}

	labels := model.LabelSet{}

	// Add static labels from config
	for k, v := range conf.StaticLabels {
		labels[k] = v
	}

	// Add non-static labels from record
	for _, label := range conf.Labels {
		if val, ok := record[label]; ok {
			sanitizedKey := model.LabelName(keyReplacer.Replace(string(label)))
			if sanitizedKey.IsValid() {
				lv := model.LabelValue(fmt.Sprintf("%v", val))
				if lv.IsValid() {
					labels[sanitizedKey] = lv
				} else {
					log.Infof("Invalid value: %v", lv)
				}
			} else {
				log.Infof("Invalid label: %v", sanitizedKey)
			}
		}
	}

	// Remove labels and configured ignore list from record
	ignoreList := append(conf.IgnoreList, conf.Labels...)
	for _, label := range ignoreList {
		delete(record, label)
	}

	js, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(record)
	if err != nil {
		return err
	}
	if conf.PrintOutput {
		fmt.Println(string(js))
	}
	return lokiClient.Handle(labels, timestamp, string(js))
}
