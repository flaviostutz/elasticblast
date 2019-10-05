package main

import (
	"flag"
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	blastURL0 := flag.String("blast-url", "", "Blast URL to which REST calls will be sent after converted from ES request. Example: http://blast:6000")
	logLevel := flag.String("log-level", "info", "debug, info, warning or error")
	flag.Parse()

	switch *logLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
		break
	case "warning":
		logrus.SetLevel(logrus.WarnLevel)
		break
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
		break
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	logrus.Debug("Preparing options")

	if blastURL0* == "" {
		logrus.Error("--blast-url is required")
		os.Exit(1)
	}

	logrus.Infof("====Starting Elasticsearch to Blast bridge====")

	h := NewHTTPServer(blastURL0*)
	err := h.Start()
	if err != nil {
		logrus.Errorf("Failed to initialize bridge. err=%s", err)
		os.Exit(1)
	}
}
