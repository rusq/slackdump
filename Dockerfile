FROM golang:1.21

WORKDIR /build

COPY . .


RUN go test ./...
