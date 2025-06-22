FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy && go mod vendor

COPY . .

RUN go build

ENTRYPOINT ["./csv_gen"]
