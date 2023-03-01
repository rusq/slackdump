SHELL=/bin/sh

FILES=$(wildcard *.json.gz) $(wildcard *.state) $(wildcard C*) $(wildcard D*)

restore:
	7z x backup/backup.7z

.PHONY: restore

clean: $(FILES)
	-rm -rf $^ 
.PHONY: clean

