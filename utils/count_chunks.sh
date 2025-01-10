#!/bin/sh

gzcat $1.json.gz| jq '.t' | awk '{count[$1]++}END{for(t in count)print t,count[t]}'