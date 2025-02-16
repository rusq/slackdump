#!/bin/sh
mkdir -p ~/.cache/slackdump/
echo $ORA600_CREDS |base64 -d > ~/.cache/slackdump/ora600.bin
echo ora600 > ~/.cache/slackdump/workspace.txt
md5sum ~/.cache/slackdump/ora600.bin
go build ./cmd/slackdump
./slackdump workspace list
./slackdump workspace select ora600
./slackdump workspace list
./slackdump list channels -no-encryption
