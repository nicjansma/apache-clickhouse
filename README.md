# apache-clickhouse &nbsp; [![Tweet](https://img.shields.io/twitter/url/http/shields.io.svg?style=social)](https://twitter.com/intent/tweet?text=Simple%20Apache%20logs%20parser%20and%20transporter%20to%20ClickHouse%20database.%20&amp;url=https://github.com/nicjansma/apache-clickhouse&amp;hashtags=apache,clickhouse,golang)

[![License: Apache 2](https://img.shields.io/hexpm/l/plug.svg)](https://github.com/nicjansma/apache-clickhouse/blob/master/LICENSE)
![Golang Version](https://img.shields.io/badge/golang-1.5%2B-blue.svg)
[![Docker](https://img.shields.io/docker/v/nicjansma/apache-clickhouse)](https://hub.docker.com/r/nicjansma/apache-clickhouse/)
[![Docker Pulls](https://img.shields.io/docker/pulls/nicjansma/apache-clickhouse.svg)](https://hub.docker.com/r/nicjansma/apache-clickhouse/)
[![Docker Stars](https://img.shields.io/docker/stars/nicjansma/apache-clickhouse.svg)](https://hub.docker.com/r/nicjansma/apache-clickhouse/)
[![GitHub issues](https://img.shields.io/github/issues/nicjansma/apache-clickhouse.svg)](https://github.com/nicjansma/apache-clickhouse/issues)

Apache, nginx, CloudFront and S3 logs parser &amp; transporter to ClickHouse databases, based on [nginx-clickhouse](https://github.com/mintance/nginx-clickhouse).

## Background

For the past 20+ years, I've been generating traffic reports based on my web server's [Apache Access Logs](https://httpd.apache.org/docs/current/logs.html), using open-source tools like [awstats](https://awstats.sourceforge.io/) and [jawstats](https://github.com/webino/JAWStats).

Development of those projects has tapered off, and their UIs are not very modern or efficient (storing pre-computed stats in text files).  I've been searching for a way of modernizing my log telemetry, and eventually settled on building a data warehouse in [ClickHouse](https://clickhouse.com/) and [Grafana](https://grafana.com/).

I recently found the [nginx-clickhouse](https://github.com/mintance/nginx-clickhouse) project which imports nginx access logs (which are essentially the same as Apache access logs) into ClickHouse.  I wanted a bit more flexability to analyze Apache, Amazon CloudFront, and Amazon S3 logs, so have forked the project for my needs.

## Changes

Improvemnts from [nginx-clickhouse](https://github.com/mintance/nginx-clickhouse):

* Process logs from:
  * Apache Access Logs
  * nginx Access Logs
  * Amazon CloudFront Access Logs
  * Amazon S3 Access Logs
  * (+any other log that can be processed with simple regexs)
* Ability to run `-once` on a file and exit (bulk loading historical data)
* Ability to read from `-stdin`
* Ability to specify the `-domain xyz.com` from the command line and/or `config.yml`
* Flexible ClickHouse column definitions and custom parsing based on the column name
* Apache, nginx, CloudFront and S3-focused Grafana dashboards
* Integration of [ua-parser](https://github.com/ua-parser/uap-go) to provide Browser and OS stats
* Integration of [crawlerdetect](https://github.com/x-way/crawlerdetect) and [isbot](https://pkg.go.dev/zgo.at/isbot) to detect bots
* Integration of [maxmind](https://github.com/oschwald/maxminddb-golang) to determine country

## How to Build From Sources

### Linux / MacOS

```sh
make build
```

### Windows

```cmd
go build -a -o apache-clickhouse.exe
```

## How to Build a Docker Image

To build image just type the command below, and it will compile binary from sources and create Docker image.

You don't need to have Go development tools, the [build process will be in Docker](https://medium.com/travis-on-docker/multi-stage-docker-builds-for-creating-tiny-go-images-e0e1867efe5a).

```sh
make docker
```

## How to Run

## Preparing Additional Data

### MaxMind Country Database

apache-clickhouse utilizes the [free MaxMind GeoLite2 Country database](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) to tag countries based on the IP address.

The MaxMind download can be put in to `data\GeoLite2-Country.mmdb` or as `apache-clickhouse.geolite2-country.mmdb`.

### ua-parser Definitions

apache-clickhouse utilizes the [ua-parser `regexs.yaml`](https://github.com/ua-parser/uap-core/blob/master/regexes.yaml) to break down the `User-Agent` string into Browser and OS names and Major versions.

The ua-parser YAML download can be put in to `data\uaparser.yml` or as `apache-clickhouse.uaparser.yml`.

### Running

By default, apache-clickhouse will monitor the specified access log, and import any new lines to ClickHouse on a regular basis (specified by `settings.interval`).  This should be used for live server logs (e.g. still being appended to).

Alternatively, you can have apache-clickhouse run `-once` (or `settings.once=true`) to parse a file and exit.  This can be used to bulk load historical data (e.g. files that are no longer changing).

This project assumes you'll be loading in data for multiple domains into the same ClickHouse tables, so you can specify `-domain [domain.xyz]` in the command line to differentiate the source of each log.  If `clickhouse.columns.domain` is missing from the `config.yml` this isn't necessary.

#### Run from Executable

```sh
apache-clickhouse -config_path [config.yml or path] [-once] -log_path [path] -domain [test.com]
```

#### Run from Docker

In the container:

* `/apache-clickhouse` is the binary
* `/config.yml` is the default config file location
* `/logs/access_log` is the default log to read from

These can be changed by running with a different command line or environment variables.

```sh
docker pull nicjansma/apache-clickhouse

docker run --rm --name apache-clickhouse -v ${PWD}/logs:/logs -v ${PWD}/config.yml:/config.yml nicjansma/apache-clickhouse

# or full command
docker run --rm --name apache-clickhouse -v ${PWD}/logs:/logs -v ${PWD}/config.yml:/config.yml nicjansma/apache-clickhouse /apache-clickhouse -config_path /config.yml -log_path /logs/access_log
```

## Configuration

The configuration is specified in `config.yml` or via an alternative file on the command line as `-config [path.yml]`.

A sample configuration file is provided in `config-sample.yml` in this repository.

### Log Formats

Each log type will need a different `log.format` specified in `config.yml`.  Example formats can be used from `config-sample.yml`.

#### Apache

Apache access logs are configured in the Apache config via the `LogFormat` directive.  The [Combined Log Format](https://httpd.apache.org/docs/current/logs.html) is commonly used, though other formats should work via updated regular expression rules in `config.yml`.

Example:

```text
LogFormat "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\"" combined
CustomLog log/access_log combined
```

#### nginx

In nginx, [nginx_http_log_module](http://nginx.org/en/docs/http/ngx_http_log_module.html) configures request logs.

Example:

```lua
http {
    ...
     log_format main '$remote_addr - $remote_user [$time_local] "$request" $status $bytes_sent "$http_referer" "$http_user_agent"';
    ...
}
```

The site then specifies the `access_log` using this `main` format:

```lua
server {
  ...
  access_log /var/log/nginx/my-site-access.log main;
  ...
}
```

#### Amazon CloudFront

[Amazon CloudFront Access Logs](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/AccessLogs.html) can be configured via the Amazon Developer console.

The standard [CloudFront log format](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/AccessLogs.html#LogFileFormat) can be parsed, though `log.optional_fields=true` is suggested in the `config.yml` to allow for fields that may be added or removed over time.

#### Amazon S3

[Amazon S3 Access Logs](https://docs.aws.amazon.com/AmazonS3/latest/userguide/ServerLogs.html) can be configured via the Amazon Developer console.

The standard [S3 log format](https://docs.aws.amazon.com/AmazonS3/latest/userguide/LogFormat.html) can be parsed, though `log.optional_fields=true` is suggested in the `config.yml` to allow for fields that may be added or removed over time.

#### Other

If logs from other applications or services are formatted with space, tab or CSV/TSV deliminators, it should be possible for this project to parse them.

### ClickHouse Table Schema

Each supported log type may have different fields and those fields can be mapped to columns in ClickHouse.

Below are some suggested schemas for each log type.

#### Apache Access Logs

```sql
CREATE TABLE metrics.apache_logs (
  domain LowCardinality(String),
  remote_addr IPv4,
  time_local DateTime,
  date Date DEFAULT toDate(current_timestamp()),
  method LowCardinality(String),
  url String,
  url_extension LowCardinality(String),
  http_version LowCardinality(String),
  status UInt16,
  body_bytes_sent UInt32,
  referrer_domain String,
  user_agent_family LowCardinality(String),
  user_agent_major LowCardinality(String),
  os_family LowCardinality(String),
  os_major LowCardinality(String),
  device_family LowCardinality(String),
  country LowCardinality(String),
  bot Boolean
) ENGINE = MergeTree()
PARTITION BY (domain, toYYYYMM(date))
ORDER BY (domain, date, status)
;
```

#### nginx Access Logs

```sql
CREATE TABLE metrics.nginx_logs (
  domain LowCardinality(String),
  remote_addr IPv4,
  time_local DateTime,
  date Date DEFAULT toDate(current_timestamp()),
  method LowCardinality(String),
  url String,
  url_extension LowCardinality(String),
  http_version LowCardinality(String),
  status UInt16,
  body_bytes_sent UInt32,
  referrer_domain String,
  user_agent_family LowCardinality(String),
  user_agent_major LowCardinality(String),
  os_family LowCardinality(String),
  os_major LowCardinality(String),
  device_family LowCardinality(String),
  country LowCardinality(String),
  bot Boolean
) ENGINE = MergeTree()
PARTITION BY (domain, toYYYYMM(date))
ORDER BY (domain, date, status)
;
```

#### Amazon CloudFront Access Logs

```sql
CREATE TABLE metrics.cloudfront_logs (
  domain LowCardinality(String),
  remote_addr IPv6,
  time_local DateTime,
  date Date DEFAULT toDate(current_timestamp()),
  cluster LowCardinality(String),
  distribution LowCardinality(String),
  protocol LowCardinality(String),
  ssl_protocol LowCardinality(String),
  ssl_cipher LowCardinality(String),
  http_host LowCardinality(String),
  method LowCardinality(String),
  url String CODEC(ZSTD),
  url_extension LowCardinality(String),
  http_version LowCardinality(String),
  status UInt16,
  response_status LowCardinality(String),
  body_bytes_sent UInt32,
  request_bytes_received UInt32,
  content_type LowCardinality(String),
  duration UInt16,
  referrer_domain String,
  user_agent_family LowCardinality(String),
  user_agent_major LowCardinality(String),
  os_family LowCardinality(String),
  os_major LowCardinality(String),
  device_family LowCardinality(String),
  country LowCardinality(String),
  bot Boolean
) ENGINE = MergeTree()
PARTITION BY (domain, toYYYYMM(date))
ORDER BY (domain, date, status)
```

#### Amazon S3 Access Logs

```sql
CREATE TABLE metrics.s3_logs (
  bucket LowCardinality(String),
  time_local DateTime,
  date Date DEFAULT toDate(current_timestamp()),
  remote_addr IPv6,
  ssl_protocol LowCardinality(String),
  ssl_cipher LowCardinality(String),
  http_host LowCardinality(String),
  operation LowCardinality(String),
  method LowCardinality(String),
  url String CODEC(ZSTD),
  url_extension LowCardinality(String),
  http_version LowCardinality(String),
  status UInt16,
  body_bytes_sent UInt32,
  duration UInt16,
  error_code LowCardinality(String),
  referrer_domain String,
  user_agent_family LowCardinality(String),
  user_agent_major LowCardinality(String),
  os_family LowCardinality(String),
  os_major LowCardinality(String),
  device_family LowCardinality(String),
  country LowCardinality(String),
  bot Boolean
) ENGINE = MergeTree()
PARTITION BY (bucket, toYYYYMM(date))
ORDER BY (bucket, date, status)
```

### Config file

#### 1. Log path & flushing interval

```yaml
settings:
  interval: 5 # in seconds
  log_path: access_log # path to logfile
  seek_from_end: false # start reading from the last line (to prevent duplicates after restart)
  once: false # whether to read the file once and exit
  domain: test.com # domain name to use
  debug: false # debug log level
```

#### 2. ClickHouse credentials and table schema

```yaml
clickhouse:
  db: metrics # Database name
  table: apache_logs # Table name
  host: localhost # ClickHouse host
  port: 8123 # ClicHhouse HTTP port
  credentials:
    user: default # User name
    password: # User password
```

Based on the chosen log format, you may want different columns parsed from the log and set in the ClickHouse table.

The `config-sample.yml` has suggest columns for each log format.

Only the columns set in `clickhouse.columns` will be sent to ClickHouse.

```yaml
columns:
  #
  # Apache
  #
  - domain
  - remote_addr
  - remote_user
  - time_local
  - date
  - method
  - url
  - url_extension
  - http_version
  - status
  - body_bytes_sent
  - referrer
  - referrer_domain
  - user_agent
  - user_agent_family
  - user_agent_major
  - os_family
  - os_major
  - device_family
  - country
  - bot
```

#### 3. Log format

The log format defines how the log will be parsed.

Examples for Apache, nginx, CloudFront and S3 are in the `config-sample.yml`.

```yaml
# Apache
log:
  format: $remote_ip - $remote_user [$time_local] "$method $url" $status $bytes "$http_referer" "$http_user_agent"
```

For other log formats, it's possible the log parser will work by specifying any fields with `$variable_name` as per above.  The [gonx](https://github.com/satyrius/gonx) parser is used, which converts those variables to regular expressions for extraction.  By default, all unkown fields will be imported as strings.

## Grafana Dashboard

Example grafana dashboards are available in the `grafana-dashboards/` folder.

![Grafana dashboard](https://github.com/nicjansma/apache-clickhouse/blob/master/grafana-dashboards/apache-access-logs.png)

## Thanks

Thanks to the [nginx-clickhouse](https://github.com/mintance/nginx-clickhouse) project for providing the starting point.
