#!/bin/bash
#
# dump.sh
#
# Dumps messages and attachments for selected 1-1 direct messages, and selected named
# channels and group PMs, from the authenticated Slack workspace. Subsequent runs will
# fetch only the new content since the previous run.
#
# NOTE: This will cache the user and channel listing, so if new users or channels are
# expected it is best to delete these files so they are re-acquired.
#
# Usage:
#
# 1. Get slackdump setup and authenticated (see https://github.com/rusq/slackdump )
# 2. Update the INCLUDE_USER_NAME_REGEX with patterns of user names whose 1-1
#    conversations should be dumped.
# 3. Update the INCLUDE_GROUP_CHANNEL_NAME_REGEX with patterns of group channels to dump.
#    NOTE: You can comment out the 'dump_list' line at the end and call 'channel_names'
#          instead, which will list all group channels (not 1-1 or PM groups) to aid you.
# 4. Ensure this script is located in the same directory as the 'slackdump' binary.
# 5. Execute the script from a terminal via './dump.sh'
#
# Dependencies:
# https://github.com/rusq/slackdump
# https://stedolan.github.io/jq/
#
# Levi Brown (@levigroker)
# v1.0.1 2023-03-07
# https://gist.github.com/levigroker/fa7b231373e68269843aeeee5cc845a3
##

### Configuration

OUTPUT_DIR="./out"
USER_LIST_FILE="${OUTPUT_DIR}/_users.json"
CHANNEL_LIST_FILE="${OUTPUT_DIR}/_channels.json"

# An array of regular expressions used to match the desired 1-1 direct messages to archive
# NOTE: Use '.*' as the pattern to match all 1-1 direct message conversations
INCLUDE_USER_NAME_REGEX=(
'.*'
)

# An array of regular expressions used to match the desired channels to archive
# NOTE: Use '^mpdm-.+' as the pattern to match all direct group conversations
INCLUDE_GROUP_CHANNEL_NAME_REGEX=(
'^mpdm-.+'
)

### Qualify needed binaries
JQ_B=$(which jq)
[ $? -ne 0 ] && echo "Please be sure jq is available in your PATH. https://stedolan.github.io/jq/" && exit 1
SLACKDUMP_B="./slackdump"
[ ! -x "$SLACKDUMP_B" ] && echo "Please be sure slackdump is located at '$SLACKDUMP_B'. https://github.com/rusq/slackdump" && exit 1

### Functions

function get_users {
	local LOG_FILE="${OUTPUT_DIR}/_log.txt"
	$SLACKDUMP_B -list-users -r json -o "${USER_LIST_FILE}" > "$LOG_FILE" 2>&1
	# Remove the logfile
	rm -f "$LOG_FILE"
}

function get_list {
	local LOG_FILE="${OUTPUT_DIR}/_log.txt"
	$SLACKDUMP_B -list-channels -r json -o "${CHANNEL_LIST_FILE}" > "$LOG_FILE" 2>&1
	# Remove the logfile
	rm -f "$LOG_FILE"
}

# Echo out all channel names. This excludes any direct message groups and 1-1 messages.
function channel_names {
	echo "$CHANNEL_LIST_JSON" | $JQ_B -r '[ map(select( (.name_normalized != "") and (.name_normalized | startswith("mpdm-") | not) ) ) | .[] | .name_normalized ] | sort | .[]'
}

# Echo out all 1-1 direct message channels as user_name channel_id pairs
function im_channels {
	# Get a list of all 1-1 direct messages
	local LIST=()
	LIST+=($(echo "$CHANNEL_LIST_JSON" | $JQ_B -r 'map(select(.is_im == true)) | .[] | .user, .id'))
	# Map the 1-1 direct message channel ID to user name
	while [ ${#LIST[@]} -gt 1 ]; do
		local USER_ID=${LIST[0]}
		local CHANNEL_ID=${LIST[1]}
		LIST=( "${LIST[@]:2}" )
		local USER_NAME=$(echo "$USER_LIST_JSON" | $JQ_B -r "map(select(.id == \"${USER_ID}\")) | .[].name")
		# Filter the list by username, as per the configured INCLUDE_USER_NAME_REGEX list 
		for NAME_REGEX in "${INCLUDE_USER_NAME_REGEX[@]+"${INCLUDE_USER_NAME_REGEX[@]}"}"; do
			if [[ "${USER_NAME}" =~ $NAME_REGEX ]]; then
				# If the regex is wide open ('.*') we may encounter empty user names, so
				# default to the user ID if we don't have a user name
				if [ "${USER_NAME}" = "" ]; then
					USER_NAME="${USER_ID}"
				fi
				echo "@${USER_NAME}" "${CHANNEL_ID}"
				break
			fi
		done
	done
}

# Echo out all group message channels as channel_name channel_id pairs
function group_channels {
	# Get a list of channel name and channel id pairs from the array of channel name regex patterns
	for CHANNEL_REGEX in "${INCLUDE_GROUP_CHANNEL_NAME_REGEX[@]+"${INCLUDE_GROUP_CHANNEL_NAME_REGEX[@]}"}"; do
		echo "$CHANNEL_LIST_JSON" | $JQ_B -rc "map(select(.name_normalized | test(\"$CHANNEL_REGEX\"))) | .[] | .name_normalized, .id"
	done
}

function dump {
	local CHANNEL_NAME="$1"
	local CHANNEL_ID="$2"
	echo "***** Dumping '$CHANNEL_NAME' ('$CHANNEL_ID')"
	
	## Handle metadata
	local META_FILE="${OUTPUT_DIR}/$CHANNEL_NAME/_meta.json"
	# Create a blank meta file if needed
	if [ ! -r "$META_FILE" ]; then
		echo "No metadata file at \"$META_FILE\". Creating it..."
		mkdir -p "${OUTPUT_DIR}/$CHANNEL_NAME"
		echo "{}" > "$META_FILE"
	fi
	# Read the meta into memory
	local META_JSON=$(<"$META_FILE")

	local PREVIOUS_DATE="$(echo "$META_JSON" | $JQ_B -r '.last_updated | select(. != null)')"
	local FROM_FLAG=""
	if [ "$PREVIOUS_DATE" != "" ]; then
		FROM_FLAG="-dump-from $PREVIOUS_DATE"
	fi
	
	local TO_FLAG="-dump-to $CURRENT_DATE"
	
	# slackdump flags to control time period:
	# 	-dump-from timestamp
	# 		timestamp of the oldest message to fetch from (i.e. 2020-12-31T23:59:59)
	# 	-dump-to timestamp
	# 		timestamp of the latest message to fetch to (i.e. 2020-12-31T23:59:59)
	# See https://github.com/rusq/slackdump/discussions/193
	
	local BASE_DIR="${OUTPUT_DIR}/${CHANNEL_NAME}"
	local LOG_FILE="${BASE_DIR}/_log.txt"
	local CHANNEL_FILE="${BASE_DIR}/${CHANNEL_ID}.json"
	local CHANNEL_FILE_OLD="${BASE_DIR}/${CHANNEL_ID}_old.json"
	# Move any pre-existing channel file out of the way so it does not get clobbered
	if [ -r "$CHANNEL_FILE" ]; then
		mv "$CHANNEL_FILE" "$CHANNEL_FILE_OLD"
	fi

	echo "Dumping messages from \"$PREVIOUS_DATE\" to \"$CURRENT_DATE\""
 	$SLACKDUMP_B -download -r json $FROM_FLAG $TO_FLAG -base "$BASE_DIR" "$CHANNEL_ID" > "$LOG_FILE" 2>&1
 	
 	local NEW_MESSAGE_COUNT=$($JQ_B -r '.messages | length' "$CHANNEL_FILE")
 	echo "Found '$NEW_MESSAGE_COUNT' new message(s)."

	# If we have an old file...
	if [ -r "$CHANNEL_FILE_OLD" ]; then
		# If there are new messages, merge the old channel messages with the new messages
		# and remove the old file
		if [ $NEW_MESSAGE_COUNT -gt 0 ]; then
			# See https://stackoverflow.com/a/75597380/397210
			local MERGED_CONTENT=$($JQ_B -s '.[0] as $o1 | .[1] as $o2 | ($o1 + $o2) | .messages = ($o1.messages + $o2.messages)' "$CHANNEL_FILE_OLD" "$CHANNEL_FILE")
			echo "$MERGED_CONTENT" > "$CHANNEL_FILE"
			rm -f "$CHANNEL_FILE_OLD"
		else
			# If there are no new messages, then delete the new file and keep the old
			rm -f "$CHANNEL_FILE"
			mv "$CHANNEL_FILE_OLD" "$CHANNEL_FILE"
		fi
	fi
	local TOTAL_MESSAGE_COUNT=$($JQ_B -r '.messages | length' "$CHANNEL_FILE")
	echo "Total messages for '$CHANNEL_NAME' channel: $TOTAL_MESSAGE_COUNT"

	# Update the last updated date
	META_JSON="$(echo "$META_JSON" | $JQ_B -r ".last_updated = \"$CURRENT_DATE\"")"
	# Persist the metadata
	echo "$META_JSON" > "$META_FILE"
	# Remove the logfile
	rm -f "$LOG_FILE"
	
	echo "*****"
}

function dump_list {
	local LIST=()
	# Add all 1-1 direct message channels
	LIST+=( $(im_channels) )
	# Add all group message channels
	LIST+=( $(group_channels) )

	# Iterate over each matching channel and archive the channel contents
	while [ ${#LIST[@]} -gt 1 ]; do
		CHANNEL_NAME=${LIST[0]}
		CHANNEL_ID=${LIST[1]}
		LIST=( "${LIST[@]:2}" )
		dump "${CHANNEL_NAME}" "${CHANNEL_ID}"
	done
}

### Start

# Fail on error, and unassigned variable use
set -eu

# Create the output directory
mkdir -p "${OUTPUT_DIR}"

# Get the user listing, if we don't already have it
if [ ! -f "${USER_LIST_FILE}" ]; then
	echo "***** Fetching user listing..."
	get_users
fi

# Get the full channel listing, if we don't already have it
if [ ! -f "${CHANNEL_LIST_FILE}" ]; then
	echo "***** Fetching channel listing..."
	get_list
fi

# Read the channel listing into memory
CHANNEL_LIST_JSON=$(<"$CHANNEL_LIST_FILE")
# Read the user listing into memory
USER_LIST_JSON=$(<"$USER_LIST_FILE")

# Capture current date/time in "2020-12-31T23:59:59" format (in UTC time zone)
CURRENT_DATE="$(date -u "+%Y-%m-%dT%H:%M:%S")"

# Dump all matching channels to the output directory as zip archives named by channel name
dump_list

# List all group channel names (not 1-1 or PM groups)
#channel_names
