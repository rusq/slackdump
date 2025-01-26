#!/bin/sh

if [ -z "$1" ] ; then
	echo Usage: $0 chunk_dir
	exit 2
fi

avg() {
	awk '{ t+=$0;n++;} END { printf("total: %d, count: %d, average: %f\n",t,n,t/n);}'
}

p75() {
	awk '{ a[i++]=$0; } END { x=int((i+1)*0.75); if (x < (i+1)*0.75) print "P75:" (a[x-1]+a[x])/2; else print "P75:" a[x-1]; }'
}

tmpfile=$(mktemp)
(

dir=$1
cd ${dir}
for f in *.json.gz; do
	gzcat $f | wc -l
done | sort > $tmpfile
)

cat $tmpfile | avg
cat $tmpfile | p75
rm $tmpfile