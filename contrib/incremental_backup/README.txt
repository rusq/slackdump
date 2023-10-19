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