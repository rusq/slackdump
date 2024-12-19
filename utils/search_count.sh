#!/bin/sh

if [ -d "$1" ] ; then
    archive="$1"
else
    echo "Usage: $(basename $0) <search_archive>"
    exit 1
fi

filename="${archive}/search.json.gz"

if [ ! -f "${filename}" ]; then
    echo "this script requires a search archive to work"
    exit 1
fi

gzcat "${filename}" | jq 'select(.t==10).sm.[].ts' | sort | uniq | wc -l
