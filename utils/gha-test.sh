#!/bin/sh
set +x
if [ -z "${ORA600_CREDS}" ]; then
  echo "ORA600_CREDS is not set"
  exit 1
fi

OUTPUT=~/.cache/slackdump/ora600.bin

mkdir -p ~/.cache/slackdump/
curl -o ~/.cache/slackdump/ora600.bin -u "${ORA600_CREDS}" --basic http://tts.endless.lol:12087/ora600.bin 

if [ -s ${OUTPUT} ]; then
  echo "Downloaded ora600.bin"
else
  echo "Failed to download ora600.bin"
  exit 1
fi

echo ora600 > ~/.cache/slackdump/workspace.txt
ls -l ~/.cache/slackdump/

md5sum ~/.cache/slackdump/ora600.bin


go build ./cmd/slackdump
./slackdump workspace list -v
./slackdump workspace select -v ora600
./slackdump workspace list -v
./slackdump list channels -v -no-encryption
