SRC=slackdump.go
OUTPUT=slackdump

LDFLAGS=-s -w

.PHONY: all run clean

all: $(OUTPUT)

$(OUTPUT): $(SRC)
	go build -ldflags "$(LDFLAGS)" -o $@ $^

run: $(SRC)
	go run $^ $(FLAGS)


clean:
	-rm $(OUTPUT)
