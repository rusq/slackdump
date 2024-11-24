SHELL=/bin/sh

CMD=./cmd/slackdump
OUTPUT=slackdump
EXECUTABLE=slackdump
BUILD=$(shell git describe --tags)
BUILD_DATE=$(shell TZ=UTC date +%Y-%m-%d\ %H:%M:%SZ)
COMMIT=$(shell git rev-parse --short HEAD)

PKG=github.com/rusq/slackdump/v3

LDFLAGS="-s -w -X 'main.commit=$(COMMIT)' -X 'main.version=$(BUILD)' -X 'main.date=$(BUILD_DATE)'"
OSES=linux darwin windows
DISTFILES=README.md LICENSE
ZIPFILES=$(foreach s,$(OSES),$(OUTPUT)-$s.zip)


.PHONY: dist all test

# special guest.
$(OUTPUT)-windows.zip: EXECUTABLE=$(OUTPUT).exe

$(foreach s,$(OSES),$(eval $(OUTPUT)-$s.zip: GOOS=$s))
$(foreach s,$(OSES),$(eval $(OUTPUT)-$s.zip: $(EXECUTABLE)))


all: $(EXECUTABLE)

dist:
	$(MAKE) $(ZIPFILES)

%.zip:
	7z a $@ $(EXECUTABLE) $(DISTFILES)
	rm $(EXECUTABLE)


$(OUTPUT).exe: GOOS=windows
$(OUTPUT).exe: $(OUTPUT)

$(OUTPUT):
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -ldflags=$(LDFLAGS) -o $(EXECUTABLE) $(CMD)

x86_%:
	GOARCH=amd64 go build -ldflags=$(LDFLAGS) -o $@ $(CMD)

arm_%:
	GOARCH=arm64 go build -ldflags=$(LDFLAGS) -o $@ $(CMD)


clean:
	-rm slackdump slackdump.exe $(wildcard *.zip)

test:
	go test -race -cover ./...

aurtest:
	GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -mod=readonly -modcacherw" go build -o 'deleteme' ./cmd/slackdump
	rm deleteme
.PHONY: aurtest

docker_test:
	docker build .

callvis:
	go-callvis -group pkg,type -limit $(PKG) $(PKG)/cmd/slackdump

goreleaser:
	goreleaser check
	goreleaser release --snapshot --clean

tags:
	gotags -R *.go > $@

generate: | install_tools
	go generate ./...
.PHONY:generate

install_tools:
	go install go.uber.org/mock/mockgen@latest
	go install golang.org/x/tools/cmd/stringer@latest
.PHONY: install_tools
