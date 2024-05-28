package apache

import (
	"io"
	"strings"

	"github.com/nicjansma/apache-clickhouse/config"
	"github.com/nicjansma/gonx"
	"github.com/sirupsen/logrus"
)

func GetParser(config *config.Config) (*gonx.Parser, error) {
	return gonx.NewParserWithOptional(config.Log.Format, config.Log.OptionalFields), nil
}

func ParseLogs(parser *gonx.Parser, logLines []string) []gonx.Entry {

	logReader := strings.NewReader(strings.Join(logLines, "\n"))
	reader := gonx.NewParserReader(logReader, parser)

	var logs []gonx.Entry

	for {
		rec, err := reader.Read()

		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		// Process the record... e.g.
		logs = append(logs, *rec)
	}

	logrus.Info("Parsed ", len(logs), " logs.")

	return logs
}
