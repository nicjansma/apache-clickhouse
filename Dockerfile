# build stage
FROM golang:1.22-alpine AS build-env

WORKDIR /go/src/github.com/mintance/apache-clickhouse

ADD . /go/src/github.com/mintance/apache-clickhouse

RUN apk update && apk add make g++ git curl
RUN cd /go/src/github.com/mintance/apache-clickhouse && go get . 
RUN cd /go/src/github.com/mintance/apache-clickhouse && make build

# final stage
FROM scratch

COPY --from=build-env /go/src/github.com/mintance/apache-clickhouse/apache-clickhouse /
CMD [ "/apache-clickhouse" ]

