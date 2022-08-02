FROM golang:1.18.4

WORKDIR /build

COPY . .


RUN go test ./...
