FROM golang:alpine AS stage

WORKDIR /build

# Copy go.mod/go.sum first for dependency caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV CI=true

RUN go test ./... \
    && go build -ldflags="-s -w" ./cmd/slackdump

FROM alpine:latest

COPY --from=stage /build/slackdump /usr/local/bin/slackdump

# create slackdump user
RUN addgroup -S slackdump && adduser -S slackdump -G slackdump \
   && mkdir /work && chown slackdump:slackdump /work
# switch to slackdump user
USER slackdump

WORKDIR /work

ENTRYPOINT ["/usr/local/bin/slackdump"]
