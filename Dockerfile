FROM golang:1.19

WORKDIR /build

COPY . .


RUN go test ./...
