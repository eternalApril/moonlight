FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-w -s" -o moonlight /app/cmd/server/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/moonlight .

COPY config.yml .

ENTRYPOINT ["./moonlight"]