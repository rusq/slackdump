#!/bin/ksh

ATTEMPTS=${ATTEMPTS:=50}
CMD="export -machine-id 123 -files -avatars -time-from=2025-01-23T00:00:00 -time-to=2025-01-25T00:00:00"
echo Building current version...
go build ./cmd/slackdump
if [ $? != 0 ] ; then
	echo "Failed to compile";
	exit 1;
fi
echo ""
echo Attempts: ${ATTEMPTS}

for (( i = 0; i < ${ATTEMPTS} ; i++ )) ; do
	echo Attempt `expr $i + 1` of $ATTEMPTS
	./slackdump ${CMD} -o dir${i} -trace=trace${i}.out -log=log${i}.log -v
done
