#!/bin/sh
mkdir -p ~/.cache/slackdump/
echo $ORA600_CREDS > ~/.cache/slackdump/ora600.bin
echo ora600 > ~/.cache/slackdump/workspace.txt
go build ./cmd/slackdump
./slackdump workspace select ora600
./slackdump workspace list -no-encryption -a
./slackdump tools info -no-encryption -auth
./slackdump list channels -no-encryption
