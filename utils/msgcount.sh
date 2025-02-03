#!/bin/sh

# counts the number of messages in the chunk file
for f in *.json.gz; do echo ${f}: $(gzcat $f | jq '(select(.t==0))| .m | length'); done
