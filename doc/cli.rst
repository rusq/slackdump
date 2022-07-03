============================
Command line flags reference
============================
[Index_]

.. contents::

This section provides some explanation for the supported command line
flags.

This doc may be out of date, to get the current command line flags
with a brief description, run::

  slackdump -h

Command line flags are described as of version ``v2.0.0``.

\-V
   print version and exit

\-c
   shorthand for -list-channels

\-cookie
   along with ``-t`` sets the authentication values.  Can also be set using
   ``COOKIE`` environment variable.  Must contain the value of ``d=`` cookie, or
   a cookies.txt dumped from the browser using the `Get cookies.txt Chrome
   extension`_

\-cpr
   number of conversation items per request. (default 200).  This is
   the amount of individual messages that will be fetched from Slack
   API per single API request.

\-dl-retries number
   rate limit retries for file downloads. (default 3).  If the file
   download process hits the Slack Rate Limit reponse (HTTP ERROR
   429), slackdump will retry the download this number of times, for
   each file.

\-download
   enable files download.  If this flag is specified, slackdump will
   download all attachments, including the ones in threads.

\-download-workers
   number of file download worker threads. (default 4).  File download
   is performed with multiple goroutines.  This is the number of
   goroutines that will be downloading files.  You generally wouldn't
   need to modify this value.

\-dump-from
   timestamp of the oldest message to fetch from
   (i.e. 2020-12-31T23:59:59).  Allows setting the lower boundary of
   the timeframe for conversation dump.  This is useful when you don't
   need everything from the beginning of times.

\-dump-to
   timestamp of the latest message to fetch to
   (i.e. 2020-12-31T23:59:59).  Same as above, but for upper boundary.

\-export name
   enables the mode of operation to "Slack Export" mode and sets the export
   directory to "name".  To save to a ZIP file, add .zip extension, i.e.
   ``name.zip``.

\-f
   shorthand for -download (means "files")

\-ft
   output file naming template.  This parameter allows to define
   custom naming for output conversation files.

   It uses `Go templating`_ system.  Available template tags:

   :{{.ID}}: channel ID
   :{{.Name}}: channel Name
   :{{.ThreadTS}}: thread timestamp.  This tag can not be used on it's
      own, it must be combined with at least one of the above tags.

   You can use any of the standard template functions.  The default
   value for this parameter outputs the channelID as the filename.  For
   threads, it will use channelID-threadTS.

   Below are some of the common templates you could use.

   :Channel ID and thread:
      ::

	 {{.ID}}{{if .ThreadTS}}-{{.ThreadTS}}{{end}}

      The output file will look like "``C480129421.json``" for a
      channel if channel has ID=C480129421 and
      "``C4840129421-1234567890.123456.json``" for a thread.  This is
      the default template.

   :Channel Name and thread:

      ::

	 {{.Name}}{{if .ThreadTS}}({{.ThreadTS}}){{end}}

      The output file will look like "``general.json``" for the channel and
      "``general(123457890.123456).json``" for a thread.


\-i
   Deprecated.  Use '@' to specify the file with links and IDs:  Example::

      slackdump @my_list.txt

\-limiter-boost
   same as -t3-boost. (default 120)

\-limiter-burst
   same as -t3-burst. (default 1)

\-list-channels
   list channels (aka conversations) and their IDs for export.  The
   default output format is "text".  Use ``-r json`` to output
   as JSON.

\-list-users
   list users and their IDs.  The default output format is "text".
   Use ``-r json`` to output as JSON.

\-log file
   if specified, will output all message to the ``file`` instead of the
   screen.

\-no-user-cache
   skip fetching users.  If this flag is specified, users won't be fetched
   during startup.  This disables the username resolving for the text
   output, I don't know why someone would use this flag, but it's there
   if you must.

\-npr
   chaNnels per request.  The amount of channels that will be fetched
   per API request when listing channels.  Setting it to higher value than
   100 bears no tangible outcome - Slack never returns more than 100 channels
   per request.  Greedy.

\-o filename
   output filename for users and channels.  Use '-' for standard
   output. (default "-")

\-r format
   report (output) format.  One of 'json' or 'text'. For channels and
   users - will output only in the specified format.  For messages -
   if 'text' is requested, the text file will be generated along with
   json.

\-t API_token
   Specify slack API token, (environment: ``SLACK_TOKEN``).
   This should be used along with ``--cookie`` flag.

\-t2-boost
   Tier-2 limiter boost in events per minute (affects users and
   channels APIs).

\-t2-burst
   Tier-2 limiter burst in events (affects users and
   channels APIs). (default 1)

\-t2-retries
   rate limit retries for channel listing. (affects users and channels APIs).
   (default 20)

\-t3-boost
   Tier-3 rate limiter boost in events per minute, will be added to
   the base slack tier event per minute value.  Affects conversation
   APIs. (default 120)

\-t3-burst
   allow up to N burst events per second.  Default value is
   safe. Affects conversation APIs (default 1)

\-t3-retries
   rate limit retries for conversation.  Affects conversation APIs. (default 3)

\-trace filename
   allows to specify the trace filename and enable tracing (optional).  Use this
   flag if requested by the developer.  The trace file does not contain any
   sensitive or personal identifiable information.  It will contain the slack
   workspace name and channel IDs.

\-u
   shorthand for -list-users.

\-user-cache-age
   user cache lifetime duration. Set this to 0 to disable
   cache usage. (default 4h0m0s) User cache is used to speedup consequent
   runs of slackdump.  If set to 0, fresh user list will fetched from the 
   server every time, unless ``-no-user-cache`` is set.

\-user-cache-file
   user cache filename. (default "users.json") See note
   for -user-cache-age above.

\-v
   verbose messages

[Index_]

.. _Index: README.rst
