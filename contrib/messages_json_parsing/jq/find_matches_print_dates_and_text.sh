#!/bin/sh

SAMPLE=../../sample.json

cat "${SAMPLE}" | jq '.messages[] | select(( .text != null) and (.text | test("JoInEd";"i"))) | (.ts |= (tonumber|todate)) | .ts, .text' | tr -d '"'
