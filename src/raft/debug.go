package raft

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type logTopic string
const (
	dTicker logTopic = "TICK"
	dElection logTopic = "ELEC"
	dHeartbeat logTopic = "BEAT"
	dMake logTopic = "MAKE"
	dExecuter logTopic = "EXEC"
)

// Retrieve the verbosity level from an environment variable
func getVerbosity() int {
	v := os.Getenv("VERBOSE")
	level := 0
	if v != "" {
		var err error
		level, err = strconv.Atoi(v)
		if err != nil {
			log.Fatalf("Invalid verbosity %v", v)
		}
	}
	return level
}

var debugStart time.Time
var debugVerbosity int

func init() {
	debugVerbosity = getVerbosity()
	debugStart = time.Now()

	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func (rf *Raft) Debug(topic logTopic, format string, a ...interface{}) {
	if debugVerbosity >= 1 {
		time := time.Since(debugStart).Microseconds()
		time /= 100
		server := fmt.Sprintf("S%d T%d ", rf.me, rf.term)
		prefix := fmt.Sprintf("%06d %v ", time, string(topic))
		format = prefix + server + format
		log.Printf(format, a...)
	}
}