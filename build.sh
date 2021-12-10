#!/bin/sh
OSES="windows linux darwin"

for os in $OSES; do
	make slackdump-${os}.zip
done
