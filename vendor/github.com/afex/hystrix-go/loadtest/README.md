integration app to measure behavior of circuits under load.

`go run service/main.go -statsd mystatsdhost:8125`

`ab -n 10000000 -c 10 http://localhost:8888/`