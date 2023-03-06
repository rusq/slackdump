SHELL=/bin/sh

FILES=$(wildcard *.json.gz) $(wildcard *.state) $(wildcard C*) $(wildcard D*)

restore: $(FILES)
	7z x backup/backup.7z
.PHONY: restore

clean: $(FILES)
	-rm -rf $^ 
.PHONY: clean

backup/backup.7z: $(FILES)
	7z a $@ $^

backup: backup/backup.7z
.PHONY: backup
