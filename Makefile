SHELL=/bin/sh

CMD=./cmd/slackdump
OUTPUT=slackdump
EXECUTABLE=slackdump
BUILD=$(shell git describe --tags)
BUILD_YEAR=$(shell date +%Y)

PKG=github.com/rusq/slackdump/v2

LDFLAGS="-s -w -X 'main.build=$(BUILD)' -X 'main.buildYear=$(BUILD_YEAR)'"
OSES=linux darwin windows
DISTFILES=README.rst LICENSE
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
	go test -race -cover -count=3 ./...

docker_test:
	docker build .

man: slackdump.1

slackdump.1: README.rst
	rst2man.py $< $@ --syntax-highlight=none

callvis:
	go-callvis -group pkg,type -limit $(PKG) $(PKG)/cmd/slackdump

goreleaser:
	goreleaser check
	goreleaser release --snapshot --clean

tags:
	gotags -R *.go > $@
