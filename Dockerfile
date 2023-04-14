FROM golang:1.19-alpine3.17 AS stage

WORKDIR /build

COPY . .

ENV CGO_ENABLED=0

RUN go test ./... \
    && go build -ldflags="-s -w" ./cmd/slackdump

FROM alpine:3.17

COPY --from=stage /build/slackdump /usr/local/bin/slackdump

# create slackdump user
RUN addgroup -S slackdump && adduser -S slackdump -G slackdump \
   && mkdir /work && chown slackdump:slackdump /work
# switch to slackdump user
USER slackdump

WORKDIR /work

ENTRYPOINT ["/usr/local/bin/slackdump"]
