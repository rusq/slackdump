#!/bin/sh
mkdir -p ~/.cache/slackdump/
echo $ORA600_CREDS > ~/.cache/slackdump/ora600.bin
echo ora600 > ~/.cache/slackdump/workspace.txt
md5sum ~/.cache/slackdump/ora600.bin
go build ./cmd/slackdump
./slackdump workspace select ora600
./slackdump list channels -no-encryption
