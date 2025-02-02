#!/bin/sh
# counts chunks in the file
gzcat $1.json.gz| jq '.t' | awk '{count[$1]++}END{for(t in count)print t,count[t]}'
