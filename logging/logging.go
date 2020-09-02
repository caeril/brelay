package logging

import (
	"fmt"
	"github.com/caeril/brelay/config"
	"os"
	"strings"
	"time"
)

var accessLog *os.File
var errorLog *os.File

func Init() {

	var err error

	accessLog, err = os.OpenFile(config.Get().Logging.AccessPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("ERROR: Could not open log file %s for writing\n", config.Get().Logging.AccessPath)
	}

	errorLog, err = os.OpenFile(config.Get().Logging.ErrorPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("ERROR: Could not open log file %s for writing\n", config.Get().Logging.ErrorPath)
	}

}

func log(log *os.File, line string) {

	// prepend timestamp
	line = time.Now().Format(time.RFC3339) + " :: " + line

	// all lines need to have a newline
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}

	if log != nil {
		if _, err := log.WriteString(line); err != nil {
			fmt.Printf("ERROR: Could not write to the logfile: %s\n", err.Error())
		}
	} else {
		fmt.Printf("%s", line)
	}

}

func Access(line string) {
	log(accessLog, line)
}

func Error(line string) {
	log(errorLog, line)
}

func DeInit() {
	errorLog.Close()
	accessLog.Close()
}
