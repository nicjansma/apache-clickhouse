package clickhouse

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mintance/go-clickhouse"
	"github.com/nicjansma/apache-clickhouse/config"
	"github.com/nicjansma/gonx"
	"github.com/oschwald/maxminddb-golang"
	"github.com/sirupsen/logrus"
	"github.com/ua-parser/uap-go/uaparser"
	"github.com/x-way/crawlerdetect"
	"zgo.at/isbot"
)

var clickHouseStorage *clickhouse.Conn
var lastDateTime string

var regExpTime = regexp.MustCompile(`[01]\d|2[0-3]:[0-5]\d:[0-5]\d`)

func Save(config *config.Config, logs []gonx.Entry) error {

	storage, err := getStorage(config)

	if err != nil {
		return err
	}

	rows := buildRows(config.ClickHouse.Columns, config, logs)

	query, err := clickhouse.BuildMultiInsert(
		config.ClickHouse.Db+"."+config.ClickHouse.Table,
		config.ClickHouse.Columns,
		rows,
	)

	if err != nil {
		logrus.Error(err)
		return err
	}

	return query.Exec(storage)
}

func buildRows(keys []string, runtimeConfig *config.Config, data []gonx.Entry) (rows clickhouse.Rows) {
	//
	// Executable path
	//
	ex, err := os.Executable()

	if err != nil {
		logrus.Fatal("Could not get executable path! ", err)
	}

	exPath := filepath.Dir(ex)

	//
	// ua-parser rules
	//
	var parser *uaparser.Parser = nil

	parserDataDir, err := uaparser.New(exPath + "/data/uaparser.yml")
	if err != nil {
		parserSameDir, err := uaparser.New(exPath + "/apache-clickhouse.uaparser.yml")

		if err != nil {
			logrus.Fatal("Could not read regexes.yaml: ", err)
		}

		parser = parserSameDir
	} else {
		parser = parserDataDir
	}

	//
	// MaxMind database
	//
	var maxMind *maxminddb.Reader = nil

	maxMindDataDir, err := maxminddb.Open(exPath + "/data/GeoLite2-Country.mmdb")
	if err != nil {
		maxMindSameDir, err := maxminddb.Open(exPath + "/apache-clickhouse.geolite2-country.mmdb")

		if err != nil {
			logrus.Fatal("Could not read MaxMind: ", err)
		}

		maxMind = maxMindSameDir
	} else {
		maxMind = maxMindDataDir
	}

	defer maxMind.Close()

	// Loop through all of the input data
	for _, logEntry := range data {
		row := clickhouse.Row{}

		logrus.Debug(logEntry)

		// pre-calculate the User Agent fields
		var userAgent = CalculateColumn("user_agent", runtimeConfig, logEntry, nil, nil)
		userAgentStr := fmt.Sprintf("%v", userAgent)

		// decode UA as needed (e.g. CloudFront URL-escapes it)
		if strings.Index(userAgentStr, "%") != -1 {
			userAgentStrDec, decErr := url.QueryUnescape(userAgentStr)

			if decErr == nil {
				userAgentStr = userAgentStrDec

				logEntry.SetField(KeyForColumn("user_agent"), userAgentStr)
			}
		}

		// parse the UA via ua-parser
		uaClient := parser.Parse(userAgentStr)

		for _, column := range keys {
			row = append(row, CalculateColumn(column, runtimeConfig, logEntry, uaClient, maxMind))
		}

		logrus.Debug(row)

		rows = append(rows, row)
	}

	return rows
}

func KeyForColumn(key string) string {
	switch key {
	case "referrer",
		"referrer_domain":
		return "http_referer"

	case "url_extension":
		return "url"

	case "user_agent",
		"user_agent_family",
		"user_agent_major",
		"os_family",
		"os_major",
		"device_family",
		"bot":
		return "http_user_agent"

	case "date":
		return "time_local"

	case "country":
		return "remote_addr"

	default:
		return key
	}
}

func CalculateColumn(key string, runtimeConfig *config.Config, logEntry gonx.Entry, uaClient *uaparser.Client, maxMind *maxminddb.Reader) interface{} {

	// pre-capture the field, though this may change later
	value, valueErr := logEntry.Field(KeyForColumn(key))

	switch key {
	case "domain":
		return runtimeConfig.Settings.Domain

	case "remote_addr":
		// pseudo-parse to make sure it's somewhat valid
		if strings.Index(value, ":") == -1 &&
			strings.Index(value, ".") == -1 {
			// invalid IP
			return "0.0.0.0"
		}

		return value

	case "remote_user",
		"method",
		"url",
		"http_version",
		"request",
		"referrer":
		if valueErr != nil || value == "" {
			return ""
		}

		return value

	case "time_local":
		if value != "" {
			t, err := time.Parse(config.ApacheTimeLayout, value)

			if err != nil {
				if lastDateTime != "" {
					// return the last successful
					t, err := time.Parse(config.ApacheTimeLayout, lastDateTime)
					if err != nil {
						return ""
					}

					return t.Format(config.CHTimeLayout)
				}

				// no last good date, this may throw an error
				return ""
			}

			// save the last good date for later
			lastDateTime = value

			return t.Format(config.CHTimeLayout)
		} else {
			//
			// try to get from standalone field
			//
			value, valueErr := logEntry.Field("date")
			if value == "" || valueErr != nil {
				return lastDateTime
			}

			valueTime, valueTimeErr := logEntry.Field("time")
			if valueTime == "" || valueTimeErr != nil || !regExpTime.MatchString(valueTime) {
				return lastDateTime
			}

			// save the last good date for later
			lastDateTime = value + " " + valueTime

			return lastDateTime
		}

	case "date":
		if value != "" {
			//
			// get from time_local
			//
			t, err := time.Parse(config.ApacheTimeLayout, value)

			if err != nil {
				if lastDateTime != "" {
					// return the last successful
					t, err := time.Parse(config.ApacheTimeLayout, lastDateTime)
					if err != nil {
						return ""
					}

					return t.Format(config.CHDateLayout)
				}

				// no last good date, this may throw an error
				return ""
			}

			// save the last good date for later
			lastDateTime = value

			return t.Format(config.CHDateLayout)
		} else {
			//
			// try to get from standalone field
			//
			value, valueErr := logEntry.Field("date")
			if value == "" || valueErr != nil {
				return lastDateTime
			}

			valueTime, valueTimeErr := logEntry.Field("time")
			if valueTime == "" || valueTimeErr != nil || !regExpTime.MatchString(valueTime) {
				return lastDateTime
			}

			// save the last good date for later
			lastDateTime = value + " " + valueTime

			return value
		}

	case "body_bytes_sent",
		"request_bytes_received",
		"content_len",
		"range_start",
		"range_end",
		"ttfb",
		"c_port",
		"status":
		//
		// when error or empty, return 0
		//
		if valueErr != nil || value == "" || value == "-" {
			return 0
		}

		valNum, err := strconv.Atoi(value)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"value": value,
			}).Error("Error to convert string to int")
		}

		return valNum

	case "referrer_domain":
		if valueErr != nil || value == "" {
			return ""
		}

		u, err := url.Parse(value)
		if err != nil {
			return ""
		}

		return u.Host

	case "content_type":
		if valueErr != nil || value == "" {
			return ""
		}

		// strip anything after ;
		value := strings.Split(value, ";")[0]

		return value

	case "url_extension":
		if valueErr != nil || value == "" {
			return ""
		}

		// remove any query string first
		value := strings.Split(value, "?")[0]

		u, err := url.Parse(value)
		if err != nil || u == nil {
			return ""
		}

		return filepath.Ext(u.Path)

	case "user_agent_family":
		return uaClient.UserAgent.Family

	case "user_agent_major":
		return uaClient.UserAgent.Major

	case "os_family":
		return uaClient.Os.Family

	case "os_major":
		return uaClient.Os.Major

	case "device_family":
		return uaClient.Device.Family

	case "country":
		ip := net.ParseIP(value)

		var record struct {
			Country struct {
				ISOCode string `maxminddb:"iso_code"`
			} `maxminddb:"country"`
		}

		var errMaxMind = maxMind.Lookup(ip, &record)
		if errMaxMind != nil {
			return ""
		}

		return record.Country.ISOCode

	case "bot":
		var result = isbot.UserAgent(value)
		var resultCrawler = crawlerdetect.IsCrawler(value)

		return (isbot.Is(result) || resultCrawler)

	case "duration":
		if value == "" {
			return 0
		}

		// must be in milliseconds already
		if strings.Index(value, ".") == -1 {
			return value
		}

		// convert from decimal (seconds) to ms
		i, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return 0
		}

		return i * 1000

	default:
		return value
	}
}

func getStorage(runtimeConfig *config.Config) (*clickhouse.Conn, error) {

	if clickHouseStorage != nil {
		return clickHouseStorage, nil
	}

	cHTTP := clickhouse.NewHttpTransport()
	conn := clickhouse.NewConn(runtimeConfig.ClickHouse.Host+":"+runtimeConfig.ClickHouse.Port, cHTTP)

	params := url.Values{}
	params.Add("user", runtimeConfig.ClickHouse.Credentials.User)
	params.Add("password", runtimeConfig.ClickHouse.Credentials.Password)
	conn.SetParams(params)

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return conn, nil
}
