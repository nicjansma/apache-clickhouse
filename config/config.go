package config

import (
	"flag"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Settings struct {
		Interval    int    `yaml:"interval"`
		LogPath     string `yaml:"log_path"`
		SeekFromEnd bool   `yaml:"seek_from_end"`
		Once        bool   `yaml:"once"`
		StdIn       bool   `yaml:"stdin"`
		Domain      string `yaml:"domain"`
		Debug       bool   `yaml:"debug"`
	} `yaml:"settings"`

	ClickHouse struct {
		Db          string   `yaml:"db"`
		Table       string   `yaml:"table"`
		Host        string   `yaml:"host"`
		Port        string   `yaml:"port"`
		Columns     []string `yaml:"columns"`
		Credentials struct {
			User     string `yaml:"user"`
			Password string `yaml:"password"`
		} `yaml:"credentials"`
	} `yaml:"clickhouse"`

	Log struct {
		Format         string `yaml:"format"`
		OptionalFields bool   `yaml:"optional_fields"`
	} `yaml:"log"`
}

var configPath string
var logPath string
var once bool
var stdin bool
var domain string

var ApacheTimeLayout = "02/Jan/2006:15:04:05 -0700"
var CHTimeLayout = "2006-01-02 15:04:00"
var CHDateLayout = "2006-01-02"

func init() {
	flag.StringVar(&configPath, "config_path", "config/config.yml", "Config path.")
	flag.StringVar(&logPath, "log_path", "", "Log path.")
	flag.BoolVar(&once, "once", false, "Run once against the log then exit")
	flag.BoolVar(&stdin, "stdin", false, "Read from stdin then exit")
	flag.StringVar(&domain, "domain", "", "Domain to use")

	flag.Parse()
}

func Read() *Config {

	config := Config{}

	logrus.Info("Reading config file: " + configPath)

	var data, err = ioutil.ReadFile(configPath)

	if err != nil {
		logrus.Fatal("Config open error: ", err)
	}

	if err = yaml.Unmarshal(data, &config); err != nil {
		logrus.Fatal("Config read & unmarshal error: ", err)
	}

	// Update config with environment variables if exist
	config.SetEnvVariables()

	// Update from command line if specified
	if logPath != "" {
		config.Settings.LogPath = logPath
	}

	if once {
		config.Settings.Once = once
	}

	if stdin {
		config.Settings.StdIn = stdin
	}

	if domain != "" {
		config.Settings.Domain = domain
	}

	return &config
}

func (c *Config) SetEnvVariables() {

	// Settings

	if os.Getenv("LOG_PATH") != "" {
		c.Settings.LogPath = os.Getenv("LOG_PATH")
	}

	if os.Getenv("FLUSH_INTERVAL") != "" {

		var flushInterval, err = strconv.Atoi(os.Getenv("FLUSH_INTERVAL"))

		if err != nil {
			logrus.Errorf("error to convert FLUSH_INTERVAL string to int: %v", err)
		}

		c.Settings.Interval = flushInterval
	}

	if os.Getenv("DOMAIN") != "" {
		c.Settings.Domain = os.Getenv("DOMAIN")
	}

	if os.Getenv("DEBUG") != "" {
		c.Settings.LogPath = os.Getenv("DEBUG")
	}

	// ClickHouse

	if os.Getenv("CLICKHOUSE_HOST") != "" {
		c.ClickHouse.Host = os.Getenv("CLICKHOUSE_HOST")
	}

	if os.Getenv("CLICKHOUSE_PORT") != "" {
		c.ClickHouse.Port = os.Getenv("CLICKHOUSE_PORT")
	}

	if os.Getenv("CLICKHOUSE_DB") != "" {
		c.ClickHouse.Db = os.Getenv("CLICKHOUSE_DB")
	}

	if os.Getenv("CLICKHOUSE_TABLE") != "" {
		c.ClickHouse.Table = os.Getenv("CLICKHOUSE_TABLE")
	}

	if os.Getenv("CLICKHOUSE_USER") != "" {
		c.ClickHouse.Credentials.User = os.Getenv("CLICKHOUSE_USER")
	}

	if os.Getenv("CLICKHOUSE_PASSWORD") != "" {
		c.ClickHouse.Credentials.Password = os.Getenv("CLICKHOUSE_PASSWORD")
	}

	// Log

	if os.Getenv("LOG_FORMAT") != "" {
		c.Log.Format = os.Getenv("LOG_FORMAT")
	}

	if os.Getenv("LOG_OPTIONAL_FIELDS") != "" {
		c.Log.OptionalFields = true
	}
}
