============
Slack Dumper
============

`Buy me a cup of tea`_

`Join the discussion`_

`Read the set up guide on Medium.com`_


Purpose: dump Slack messages, users and files using browser token and cookie.

Typical use scenarios:

* archive your private conversations from Slack when the administrator
  does not allow you to install applications OR you don't want to use 
  potentially privacy-violating third-party tools, 
* archive channels from Slack when you're on a free "no archive" subscription,
  so you don't lose valuable knowledge in those channels.

The library is "fit-for-purpose" quality and provided AS-IS.  I can't
say it's ready for production, as it lacks most of the unit tests, but
will do for ad-hoc use.

Slackdump accepts two types of input: 

#. the URL/link of the channel or thread, OR 
#. the ID of the channel.

.. contents::
   :depth: 2


Usage
=====

#. Download the archive from the Releases page for your operating system. (NOTE: **MacOS users** should download ``darwin`` release file).
#. Unpack
#. Change directory to where you have unpacked the archive.
#. Run ``./slackdump -h`` to see help.

How to authenticate
-------------------

Getting the authentication data
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

#. Open up your Slack *in browser* and login.

TOKEN
+++++

#. Open your browser's *Developer Console*.
#. Go to the Network tab
#. In the toolbar, switch to ``Fetch/XHR`` view.
#. Open any channel or private conversation in Slack.  You'll see a
   bunch of stuff appearing in Network panel.
#. In the list of requests, find the one starting with
   ``channels.prefs.get?``, click it and click on *Headers* tab in the
   opened pane.
#. Scroll down, until you see **Form Data**
#. Grab the **token:** value (it starts with ``xoxc-``), by right
   clicking the value and choosing "Copy Value".

**If you don't see the token value** in Google Chrome - switch to `Payload` tab,
your token is waiting for you there.

COOKIE
++++++

**OPTION I:  Getting the cookie value**

#. Switch to Application_ tab and select **Cookies** in the left
   navigation pane.
#. Find the cookie with the name "``d``".  That's right, just the
   letter "d".
#. Double-click the Value of this cookie.
#. Press Ctrl+C or Cmd+C to copy it's value to clipboard.
#. Save it for later.

**OPTION II:  Saving cookies to a cookies.txt**

#. Install the `Get cookies.txt Chrome extension`_
#. With your Slack workspace tab opened, press the "Get Cookies.txt" extension
   button
#. Press **Export** button.
#. The slack.com_cookies.txt will have been saved to your **Downloads**
   directory.
#. Copy it to any convenient location, i.e. the directory where "slackdump"
   executable is.

Generally, there's no necessity in using the cookies.txt file, so providing
d= cookie value will work in most cases.

It may only be necessary, if your slack workspace uses Single Sign-On (SSO) in
case you keep getting ``invalid_auth`` error.

Slackdump will automatically detect if the filename is used as a value of a
cookie, and will load all cookies from that file.


Setting up the application
~~~~~~~~~~~~~~~~~~~~~~~~~~

#. Create the file named ``.env`` next to where the slackdump
   executable in any text editor.  Alternatively the file can
   be named ``secrets.txt`` or ``.env.txt``.
#. Add the token and cookie values to it. End result
   should look like this::

     SLACK_TOKEN=xoxc-<...elided...>
     COOKIE=xoxd-<...elided...>

   Alternatively, if you saved the cookies to the file, it will look like this:

     SLACK_TOKEN=xoxc-<...elided...>
     COOKIE=path/to/slack.com_cookies.txt
     
#. Save the file and close the editor.


Dumping conversations
---------------------

As it was already mentioned in the introduction, Slackdump supports
two ways of providing the conversation IDs that you want to save:

- **By ID**: it expects to see Conversation IDs.
- **By URL**: it expects to see URLs.  You can get URL by choosing
  "Copy Link" in the Slack on the channel or thread.

IDs or URLs can be passed on the command line or read from a file
(using the ``-i`` command line flag), in that file, every ID or URL
should be placed on a separate line.  Slackdump can automatically
detect if it's an ID or a URL.
  
Providing the list on the command line
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Firstly, dump the channel list to choose what you want to dump::

  slackdump -c

You will get the output resembling the following::

  2021/10/31 17:32:34 initializing...
  2021/10/31 17:32:35 retrieving data...
  2021/10/31 17:32:35 done
  ID           Arch  Saved  What
  CHXXXXXXX    -     -      #everything
  CHXXXXXXX    -     -      #everyone
  CHXXXXXXX    -     -      #random
  DHMAXXXXX    -     -      @slackbot
  DNF3XXXXX    -     -      @alice
  DLY4XXXXX    -     -      @bob

You'll need the value in the **ID** column.

To dump the channel, run the following command::

  slackdump <ID1> [ID2] ... [IDn]

By default, slackdump generates a json file with the convesation.  If
you want the convesation to be saved to a text file as well, use the
``-r text`` command line parameter.  See example below.

Example
+++++++

You want to dump conversations with @alice and @bob to text
files and save all the files (attachments) that you all shared in those
conversations::

  slackdump -r text -f DNF3XXXXX DLY4XXXXX https://....
       	    ━━━┯━━━ ━┯ ━━━┯━━━━━ ━━━┯━━━━━ ━━━━┯━━━━━┅┅ 
               │     │    │         │          │
               │     │    │         ╰─: @alice │
               │     │    ╰───────────: @bob   ┊
               │     ╰────────────────: save files
               ╰──────────────────────: text file output
           thread or conversation URL :────────╯

Conversation URL:
	       
To get the conversation URL link, use this simple trick that they
won't teach you at school:
	       
1. In Slack, right click on the conversation you want to dump (in the
   channel navigation pane on the left)
2. Choose "Copy link".

Thread URL:

1. In Slack, open the thread that you want to dump.
2. The thread opens to the right of the main conversation window
3. On the first message of the thread, click on three vertical dots menu (not sure how it's properly called), choose "Copy link"

Run the slackdump and provide the URL link as an input::

  slackdump -f  https://xxxxxx.slack.com/archives/CHM82GX00/p1577694990000400
            ━┯  ━━━━━━┯━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	     │        ╰─────: URL of the thread
	     ╰──────────────: save files
	     

Reading data from the file
~~~~~~~~~~~~~~~~~~~~~~~~~~

Slackdump can read the list of the channels and URLs to dump from the
file.

1. Create the file that will contain all the necessary IDs and/or
   URLs, I'll use "links.txt" in the example.
2. Copy/paste all the IDs and URLs into that file, one per line.
3. Run slackdump with "-i" command line flag.  "-i" stands for
   "input"::

     slackdump -i links.txt
               ━━━━┯━━━━━━━
	           │        
		   ╰───────: instructs slackdump to use the file input
		   
Dumping users
-------------

To view all users, run::

  slackdump -u

By default, slackdump exports users in text format.  If you need to
output json, use ``-r json`` flag.

Dumping channels
----------------

To view channels, that are visible to your account, including group
conversations, archived chats and public channels, run::

  slackdump -c

By default, slackdump exports users in text format.  If you need to
output json, use ``-r json`` flag.

Command line flags reference
============================

In this section there will be some explanation provided for the
possible command line flags.

This doc may be out of date, to get the current command line flags
with a brief description, run::

  slackdump -h

Command line flags are described as of version ``v1.3.1``.

\-V
   print version and exit
\-c
   same as -list-channels

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
   specify the input file with Channel IDs or URLs to be used instead
   of giving the list on the command line, one per line.  Use "-" to
   read input from STDIN.  Example: ``-i my_links.txt``.
   
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

\-o
   output filename for users and channels.  Use '-' for standard
   output. (default "-")
   
\-r
   report (output) format.  One of 'json' or 'text'. For channels and
   users - will output only in the specified format.  For messages -
   if 'text' is requested, the text file will be generated along with
   json.

\-t
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
   allows to specify the trace filename and enable tracing (optional).
   Use this flag if requested by developer.  The trace file does not contain any
   sensitive or PII.

\-u
   shorthand for -list-users.

\-user-cache-age
   user cache lifetime duration. Set this to 0 to disable
   cache. (default 4h0m0s) User cache is used to speedup consequent
   runs of slackdump.  Known issue - if you're changing slack
   workspace, make sure to delete the cache file, or set this to 0.

\-user-cache-file
   user cache filename. (default "users.json") See note
   for -user-cache-age above.

\-v
   verbose messages

As a library
============

Download:

.. code:: go

  go get github.com/rusq/slackdump

Use:

.. code:: go

  import "github.com/rusq/slackdump"

  func main() {
    sd, err := slackdump.New(os.Getenv("TOKEN"), os.Getenv("COOKIE"))
    if err != nil {
        // handle
    }
    // ... read the docs
  }

FAQ
===

:Q: **Do I need to create a Slack application?**

:A: No, you don't.  You need to grab that token and cookie from the
    browser Slack session.  See Usage_ at the top of the file.

:Q: **I'm getting "invalid_auth" error**

:A: Go get the new Cookie from the browser and Token as well.



Bulletin Board
--------------

Messages that were conveyed with the donations:

- 25/01/2022: Stay away from `TheSignChef.com`_, ya hear, they don't pay what
  they owe to their employees. 

.. _Application: https://stackoverflow.com/questions/12908881/how-to-copy-cookies-in-google-chrome
.. _`Buy me a cup of tea`: https://www.paypal.com/donate/?hosted_button_id=GUHCLSM7E54ZW
.. _`Join the discussion`: https://t.me/slackdump
.. _`Read the set up guide on Medium.com`: https://medium.com/@gilyazov/downloading-your-private-slack-conversations-52e50428b3c2
.. _`Go templating`: https://pkg.go.dev/html/template

..
  bulletin board links

.. _`TheSignChef.com`: https://www.glassdoor.com.au/Reviews/TheSignChef-com-Reviews-E793259.htm
.. _`Get cookies.txt Chrome extension`: https://chrome.google.com/webstore/detail/get-cookiestxt/bgaddhkoddajcdgocldbbfleckgcbcid
