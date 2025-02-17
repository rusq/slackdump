#!/bin/sh

SAMPLE=../../sample.json

# shellcheck disable=SC2002
cat "${SAMPLE}" | jq '.messages[]|.user +": "+.text' | tr -d '"'
