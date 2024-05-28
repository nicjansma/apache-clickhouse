build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o apache-clickhouse .

docker:
	docker build --rm -t nicjansma/apache-clickhouse -f Dockerfile .