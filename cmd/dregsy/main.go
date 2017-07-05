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

	"github.com/xelalexv/dregsy/sync"
)

//
func main() {

	config := flag.String("config", "", "path to config file")
	flag.Parse()
	if len(*config) == 0 {
		fmt.Println("synopsis: dregsy -config={config file}\n")
		os.Exit(1)
	}

	conf, err := sync.LoadConfig(*config)
	failOnError(err)

	sync, err := sync.New(conf.DockerHost, conf.APIVersion)
	failOnError(err)
	defer sync.Dispose()

	failOnError(sync.SyncFromConfig(conf))
}

//
func failOnError(err error) {
	if err != nil {
		sync.LogError(err)
		os.Exit(1)
	}
}
