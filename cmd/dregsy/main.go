/*
 * TODO:
 *	- switch to log package
 *	-
 */

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xelalexv/dregsy/internal/pkg/log"
	"github.com/xelalexv/dregsy/internal/pkg/sync"
)

var DregsyVersion string

//
func version() {
	log.Info("\ndregsy %s\n", DregsyVersion)
}

//
func main() {

	configFile := flag.String("config", "", "path to config file")
	flag.Parse()

	if len(*configFile) == 0 {
		version()
		fmt.Println("synopsis: dregsy -config={config file}\n")
		os.Exit(1)
	}

	version()

	conf, err := sync.LoadConfig(*configFile)
	failOnError(err)

	sync, err := sync.New(conf)
	failOnError(err)

	err = sync.SyncFromConfig(conf)
	sync.Dispose()
	failOnError(err)
}

//
func failOnError(err error) {
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
