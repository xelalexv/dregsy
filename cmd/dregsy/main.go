/*
	Copyright 2020 Alexander Vollschwitz <xelalex@gmx.net>

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

	  http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/sync"
)

//
func init() {

	log.SetOutput(os.Stdout)

	format := strings.ToLower(os.Getenv("LOG_FORMAT"))
	switch format {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "text":
		log.SetFormatter(&log.TextFormatter{
			ForceColors: strings.ToLower(os.Getenv("LOG_FORCE_COLORS")) == "true"})
	case "":
	default:
		log.Errorf("invalid log format: '%s'", format)
	}

	if strings.ToLower(os.Getenv("LOG_METHODS")) == "true" {
		log.SetReportCaller(true)
	}

	level := os.Getenv("LOG_LEVEL")
	if level != "" {
		l, err := log.ParseLevel(level)
		if err != nil {
			log.Errorf("invalid log level: '%s'; valid levels are: panic, "+
				"fatal, error, warn, info, debug, trace", level)
		} else {
			log.SetLevel(l)
		}
	}
}

//
var DregsyVersion string

// for invoking dregsy command during testing
var testRound bool
var testArgs []string
var testSync chan *sync.Sync
var dregsyExitCode int

//
func version() {
	log.Infof("dregsy %s", DregsyVersion)
}

//
func main() {

	dregsyExitCode = 0

	fs := flag.NewFlagSet("dregsy", flag.ContinueOnError)
	configFile := fs.String("config", "", "path to config file")

	if testRound {
		if len(testArgs) > 0 {
			failOnError(fs.Parse(testArgs))
		} else {
			panic("no test arguments")
		}
	} else {
		failOnError(fs.Parse(os.Args[1:]))
	}

	if len(*configFile) == 0 {
		version()
		fmt.Println("synopsis: dregsy -config={config file}")
		exit(1)
	}

	version()

	conf, err := sync.LoadConfig(*configFile)
	failOnError(err)

	s, err := sync.New(conf)
	failOnError(err)

	if testRound {
		testSync <- s
	}

	err = s.SyncFromConfig(conf)
	s.Dispose()
	failOnError(err)
}

//
func failOnError(err error) {
	if err != nil {
		log.Error(err)
		exit(1)
	}
}

//
func exit(code int) {
	dregsyExitCode = code
	if !testRound {
		os.Exit(code)
	}
}
