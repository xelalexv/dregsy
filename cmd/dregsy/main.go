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

//
func main() {

	configFile := flag.String("config", "", "path to config file")
	flag.Parse()

	if len(*configFile) == 0 {
		fmt.Println("synopsis: dregsy -config={config file}\n")
		os.Exit(1)
	}

	conf, err := sync.LoadConfig(*configFile)
	failOnError(err)

	sync, err := sync.New(conf)
	failOnError(err)

	defer sync.Dispose()

	failOnError(sync.SyncFromConfig(conf))
}

//
func failOnError(err error) {
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
