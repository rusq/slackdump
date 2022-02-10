============
Slack Dumper
============

`Buy me a cup of tea`_

`Join the discussion`_

`Read the set up guide on Medium.com`_

Purpose: dump slack messages, users and files using browser token and cookie.

Typical usecase scenarios:

* You want to archive your private convesations from slack but the administrator
  does not allow you to install applications.

* You are allowed to install applications in Slack but don't want to use the
  "cloud" tools for privacy concerns - god knows what those third party apps are
  retaining in their "clouds".

The library is "fit-for-purpose" quality and provided AS-IS.  Can't
say it's ready for production, as it lacks most of the unit tests, but
will do for ad-hoc use.

Slackdump accepts two types of input: URL link of the channel or
thread, or ID of the channel.

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

#. Open your browser *Developer Console*.
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

**If you don't see the token value** in Poogle Chrome - switch to `Payload` tab,
your token is waiting for you there.

COOKIE
++++++

#. Switch to Application_ tab and select **Cookies** in the left
   navigation pane.
#. Find the cookie with the name "``d``".  That's right, just the
   letter "d".
#. Double-click the Value of this cookie.
#. Press Ctrl+C or Cmd+C to copy it's value to clipboard.
#. Save it for later.

Setting up the application
~~~~~~~~~~~~~~~~~~~~~~~~~~

#. Create the file named ``.env`` next to where the slackdump
   executable in any text editor.  Alternatively the file can
   be named ``secrets.txt`` or ``.env.txt``.
#. Add the token and cookie values to it. End result
   should look like this::

     SLACK_TOKEN=xoxc-<...elided...>
     COOKIE=12345472908twp<...elided...>

#. Save the file and close the editor.


Dumping conversations
~~~~~~~~~~~~~~~~~~~~~

As it was already mentioned in the introduction, Slackdump supports
two ways of providing the conversation IDs that you want to save:

- **By ID**: it expects to see Conversation IDs.
- **By URL**: it expects to see URLs.  You can get URL by choosing
  "Copy Link" in the Slack on the channel or thread.

IDs or URLs can be passed on the command line or read from a file
(using the ``-i`` command line flag), in that file, every ID or URL
should be placed on a separate line.  Slackdump can automatically
detect the type of input.
  
Providing the list on the command line
++++++++++++++++++++++++++++++++++++++

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
^^^^^^^

Say, you want to dump convesations with @alice and @bob to the text
files and also want to save all the files that you all shared in those
convesations::

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
++++++++++++++++++++++++++

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
~~~~~~~~~~~~~

To view all users, run::

  slackdump -u

By default, slackdump exports users in text format.  If you need to
output json, use ``-r json`` flag.

Dumping chanels
~~~~~~~~~~~~~~~

To view channels, that are visible to your account, including group
conversations, archived chats and public channels, run::

  slackdump -c

By default, slackdump exports users in text format.  If you need to
output json, use ``-r json`` flag.

	       
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

Q: **Do I need to create a Slack application?**

A: No, you don't.  You need to grab that token and cookie from the
browser Slack session.  See Usage in the top of the file.

Q: **I'm getting ``invalid_auth``**

A: Go get the new Cookie from the browser.


Bulletin Board
--------------

Messages that were conveyed with the donations:

- 25/01/2022: Stay away from `TheSignChef.com`_, ya hear, they don't pay what
  they owe to their employees. 

.. _Application: https://stackoverflow.com/questions/12908881/how-to-copy-cookies-in-google-chrome
.. _`Buy me a cup of tea`: https://www.paypal.com/donate/?hosted_button_id=GUHCLSM7E54ZW
.. _`Join the discussion`: https://t.me/slackdump
.. _`Read the set up guide on Medium.com`: https://medium.com/@gilyazov/downloading-your-private-slack-conversations-52e50428b3c2

..
  bulletin board links

.. _`TheSignChef.com`: https://www.glassdoor.com.au/Reviews/TheSignChef-com-Reviews-E793259.htm
