SHELL=/bin/sh

CMD=./cmd/slackdump
OUTPUT=slackdump
EXECUTABLE=slackdump

LDFLAGS="-s -w"
OSES=linux darwin windows
DISTFILES=README.rst LICENSE
ZIPFILES=$(foreach s,$(OSES),$(OUTPUT)-$s.zip)

.PHONY: dist all

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
	GOOS=$(GOOS) go build -ldflags=$(LDFLAGS) -o $(EXECUTABLE) $(CMD)

clean:
	-rm slackdump slackdump.exe $(wildcard *.zip)
