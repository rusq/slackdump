SHELL=/bin/sh

FILES=$(wildcard *.json.gz) $(wildcard *.state) $(wildcard C*) $(wildcard D*) manual

restore: $(FILES)
	7z x backup/backup.7z
.PHONY: restore

clean: $(FILES)
	-rm -rf $^ 
.PHONY: clean

backup: $(FILES)
	7z a backup/backup.7z $^
