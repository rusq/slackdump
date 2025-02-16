#!/bin/sh
mkdir -p ~/.cache/slackdump/
curl -u $ORA600_CREDS --basic -o ~/.cache/slackdump/ora600.bin http://tts.endless.lol:120782/ora600.bin
echo ora600 > ~/.cache/slackdump/workspace.txt
md5sum ~/.cache/slackdump/ora600.bin
go build ./cmd/slackdump
./slackdump workspace list -v
./slackdump workspace select -v ora600
./slackdump workspace list -v
./slackdump list channels -v -no-encryption
