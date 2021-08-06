package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

var (
	version    = ""
	buildinfos = ""
	AppVersion = "loki-exporter " + version + " " + buildinfos
	LogLevel   = flag.String("loglevel", "info", "Log level")
	Version    = flag.Bool("v", false, "Print version")
)

func init() {
}

func main() {
	flag.Parse()

	if *Version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	lvl, _ := log.ParseLevel(*LogLevel)
	log.SetLevel(lvl)

	log.Info("Starting loki-exporter")

	// TODO
}
