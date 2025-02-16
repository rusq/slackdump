#!/bin/sh
set +x
if [ -z "${ORA600_CREDS}" ]; then
  echo "ORA600_CREDS is not set"
  exit 1
fi

OUTPUT=~/.cache/slackdump/ora600.bin
OUTPUT_DIR=$(dirname ${OUTPUT})

mkdir -p ~/.cache/slackdump/
curl -o ${OUTPUT} -u "${ORA600_CREDS}" --basic http://tts.endless.lol:12087/ora600.bin 

if [ -s ${OUTPUT} ]; then
  echo "Downloaded ora600.bin"
else
  echo "Failed to download ora600.bin"
  exit 1
fi

echo ora600 > ${OUTPUT_DIR}/workspace.txt
ls -l ${OUTPUT_DIR}

md5sum ${OUTPUT}

go build ./cmd/slackdump
./slackdump workspace select -v ora600
./slackdump workspace list -v -machine-id=123 -a
./slackdump list channels -v
