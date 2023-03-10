#!/bin/sh

SAMPLE=../../sample.json

cat "${SAMPLE}" | jq '.messages[]|.user +": "+.text' | tr -d '"'
