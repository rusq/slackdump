Dumping conversations
---------------------
[Index_]

.. contents::

As it was already mentioned in the introduction, Slackdump supports
two ways of providing the conversation IDs that you want to save:

- **By ID**: it expects to see Conversation IDs.
- **By URL**: it expects to see URLs.  You can get URL by choosing
  "Copy Link" in the Slack on the channel or thread.

IDs or URLs can be passed on the command line or read from a file
(using the ``-i`` command line flag), in that file, every ID or URL
should be placed on a separate line.

Slackdump can automatically detect if it's an ID or a URL.

Providing the list on the command line
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

By ID
+++++

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

By URL
++++++
One can start Slackdump with the list of URLs.  This can be helpful, if the
amount of channels is to large to list::

  slackdump <URL1> [URL2] ... [URLn]

One can mix URLs and IDs.

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
[Index_]

.. _Index: README.rst
.. _Issues: issues
