# build stage
FROM golang:1.22-alpine AS build-env

WORKDIR /go/src/github.com/mintance/apache-clickhouse

# update the OS
RUN apk update && apk add make g++ git curl

# add our source files
ADD . /go/src/github.com/mintance/apache-clickhouse

# make and build
RUN cd /go/src/github.com/mintance/apache-clickhouse && go get . 
RUN cd /go/src/github.com/mintance/apache-clickhouse && make build

# final stage
FROM scratch

# copy binary
COPY --from=build-env /go/src/github.com/mintance/apache-clickhouse/apache-clickhouse /

# copy support files
COPY data/uaparser.yml /apache-clickhouse.uaparser.yml
COPY data/GeoLite2-Country.mmdb /apache-clickhouse.geolite2-country.mmdb

# run command
CMD [ "/apache-clickhouse", "-config_path", "/config.yml", "-log_path", "/logs/access_log" ]

