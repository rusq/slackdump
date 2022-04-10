#!/bin/sh
OSES="windows linux darwin"

mkdir -p dist

for os in $OSES; do
  f=slackdump-${os}.zip
	make "$f"
	mv "$f" dist/
done
