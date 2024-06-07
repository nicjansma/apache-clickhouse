package main

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nicjansma/apache-clickhouse/apache"
	"github.com/nicjansma/apache-clickhouse/clickhouse"
	"github.com/nicjansma/apache-clickhouse/config"
	configParser "github.com/nicjansma/apache-clickhouse/config"
	"github.com/nicjansma/gonx"
	"github.com/papertrail/go-tail/follower"
	"github.com/sirupsen/logrus"
)

var (
	locker sync.Mutex
	logs   []string
)

func processLogs(config *config.Config, apacheParser *gonx.Parser) {
	logrus.Info("Looking for new logs to process...")

	if len(logs) == 0 {
		logrus.Info("No new logs.")
		return
	}

	if len(logs) > 0 {

		logrus.Info("Preparing to save ", len(logs), " new log entries.")
		locker.Lock()

		err := clickhouse.Save(config, apache.ParseLogs(apacheParser, logs))

		if err != nil {
			logrus.Error("Can't save logs: ", err)
		} else {
			logrus.Info("Saved ", len(logs), " new logs.")
		}

		logs = []string{}
		locker.Unlock()
	}
}

func main() {

	// Read config & incoming flags
	config := configParser.Read()

	apacheParser, err := apache.GetParser(config)

	if err != nil {
		logrus.Fatal("Can't parse apache log format: ", err)
	}

	if config.Settings.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if config.Settings.StdIn {
		logrus.Info("Reading from standard input")

		// read all of stdin
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			logs = append(logs, s.Text())
		}

		processLogs(config, apacheParser)

	} else if !config.Settings.Once {
		whenceSeek := io.SeekStart

		logrus.Info("Trying to open logfile: " + config.Settings.LogPath)

		if config.Settings.SeekFromEnd {
			whenceSeek = io.SeekEnd
		}

		t, err := follower.New(config.Settings.LogPath, follower.Config{
			Whence: whenceSeek,
			Offset: 0,
			Reopen: true,
		})

		if err != nil {
			logrus.Fatal("Can't tail logfile: ", err)
		}

		go func() {
			for {
				time.Sleep(time.Second * time.Duration(config.Settings.Interval))

				processLogs(config, apacheParser)
			}
		}()

		// Push new log entries to array
		for line := range t.Lines() {
			locker.Lock()

			logs = append(logs, strings.TrimSpace(line.String()))

			locker.Unlock()
		}

	} else {
		logrus.Info("Scanning file once: " + config.Settings.LogPath)

		logs = readFileOnce(config.Settings.LogPath)
		if logs != nil {
			processLogs(config, apacheParser)
		}
	}
}

func readFileOnce(filename string) []string {
	file, err := os.Open(filename)

	if err != nil {
		logrus.Error(err)
		return nil
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// This is our buffer now
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}
