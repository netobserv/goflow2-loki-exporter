package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/netobserv/goflow2-loki-exporter/pkg/export"
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

	var config *export.Config
	if configFile == nil || *configFile == "" {
		log.Info("Using default configuration")
		config = export.DefaultConfig()
	} else {
		flog := log.WithField("configFile", *configFile)
		flog.Info("Provided YAML config file")
		if config, err = export.LoadConfig(*configFile); err != nil {
			flog.WithError(err).Fatal("Can't load config file")
		}
	}
	loki, err := export.NewLoki(config)
	if err != nil {
		log.WithError(err).Fatal("Can't load Loki exporter")
	}
	if err := loki.Process(os.Stdin); err != nil {
		log.WithError(err).Fatal("Processing standard input")
	}
	log.Warn("Process finished")
}
