#!/bin/sh

SAMPLE=../../sample.json

# shellcheck disable=SC2002
cat "${SAMPLE}" | jq '.messages[] | select(( .text != null) and (.text | test("JoInEd";"i"))) | (.ts |= (tonumber|todate)) | .ts, .text' | tr -d '"'
