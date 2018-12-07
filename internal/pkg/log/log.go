/*
 *
 */

package log

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

//
var ToTerminal bool

func init() {
	ToTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
}

//
func Println() {
	Info("")
}

//
func Warning(msg string, params ...interface{}) {
	log("WARN", msg, params...)
}

//
func Info(msg string, params ...interface{}) {
	log("INFO", msg, params...)
}

//
func log(level, msg string, params ...interface{}) {
	msg = fmt.Sprintf(msg, params...)
	if !ToTerminal {
		msg = fmt.Sprintf(
			"%s [%s] %s", time.Now().Format(time.RFC3339), level, msg)
	}
	fmt.Print(msg)
	if !strings.HasSuffix(msg, "\n") {
		fmt.Println()
	}
}

//
func Error(err error) bool {
	if err != nil {
		if ToTerminal {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "%s [ERROR] %v\n",
				time.Now().Format(time.RFC3339), err)
		}
		return true
	}
	return false
}
