=====================
Dumping Conversations
=====================
[Index_]

.. contents::

Generic Information
-------------------
On Examples
+++++++++++

The provided examples are for Linux and macOS if you're using windows, replace
``./slackdump`` with ``slackdump`` in the examples.

User Cache
++++++++++

Slackdump always pre-caches the Slack Workspace Users to be able to resolve the
usernames.  If the amount of users is large in your Workspace, disable user
caching with ``-no-user-cache`` flag, i.e.::

  slackdump -no-user-cache ...

Output Format
+++++++++++++

The default output format for Conversations or Threads is ``json``.
Additionally, slackdump can generate a text file with formatted conversation.
To enable generation of the text file::

  slackdump -r text ...

Save to Another Directory or ZIP File
+++++++++++++++++++++++++++++++++++++

By default, Slackdump writes all files to the current directory.  Alternatively,
Slackdump can output files to another directory or even a ZIP file.  To make
Slackdump write to another directory or ZIP file:

Output to Directory with the name of "some_dir"::
  
  slackdump -base some_dir ...

Output to a ZIP file named "my_archive.zip"::

  slackdump -base my_archive.zip ...

Downloading file and image attachments
++++++++++++++++++++++++++++++++++++++

By default, Slackdump does not fetch any attachments.  To enable fetching
attachments, use ``-download`` flag::

  slackdump -download ...

If the base directory is set, it will use it to save attachments.

Using the Command Line
----------------------

To dump the conversations of interest, you must provide their IDs or URLs either
on the command line, or a file.

Providing the list on the command line::

  ./slackdump CXXXXXX DXXXXXXX https://xx.slack.com/archives/CXXXXXX

The URL can be URL of the conversation or thread.  Thread URLs are explained
in details later in this section.

Example
+++++++

You want to dump conversations with @alice and @bob to text files and save all
the files (attachments) that you all shared in those conversations::

  slackdump -r text -download DNF3XXXXX DLY4XXXXX https://....
            ━━━┯━━━ ━┯━━━━━━━ ━━━┯━━━━━ ━━━┯━━━━━ ━━━━┯━━━━━┅┅
               │     │           │         │          │
               │     │           │         ╰─: @alice │
               │     │           ╰───────────: @bob   │
               │     ╰────────────────: save files    ┊
               ╰──────────────────────: text file output (can also be "json")
                  thread or conversation URL :────────╯

Reading data from the file
--------------------------

Slackdump can read the list of the channels and URLs to dump from the
file.

1. Create the file that will contain all the necessary IDs and/or
   URLs, I'll use "links.txt" in the example.
2. Copy/paste all the IDs and URLs into that file, one per line.
3. Run slackdump with "@links.txt" on the command line::

     slackdump @links.txt
               ━━━━┯━━━━━
                   │
                   ╰───────: instructs slackdump to use the file input

   "@" character instructs slackdump to read entries from the file.

File input can be combined with channel IDs or URLs, i.e.::

  slackdump CHANNELID1 @links.txt https://xx.slack.com/...

Conversation URL
----------------

To get the conversation URL link, use this simple trick that they
won't teach you at school:

1. In Slack Client, right click on the conversation you want to dump (in the
   channel navigation pane on the left)
2. Choose "Copy link".

Thread URL
----------

1. In Slack, open the thread that you want to dump.
2. The thread opens to the right of the main conversation window
3. On the first message of the thread, click on three vertical dots menu (not
   sure how it's properly called), choose "Copy link"

Run the slackdump and provide the URL link as an input::

  slackdump -f  https://xxxxxx.slack.com/archives/CHM82GX00/p1577694990000400
            ━┯  ━━━━━━┯━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
             │        ╰─────: URL of the thread
             ╰──────────────: save files (shorthand for -download)

Internal Thread Link Format
+++++++++++++++++++++++++++
Slackdump also supports the internal format of the thread identifier for
brevity.  It has the format of CHANNEL:THREAD, i.e.
``CHM82GX00:1577694990.000400``, for the example above.

[Index_]

.. _Index: README.rst
.. _Issues: issues
.. _export: usage-export.rst
