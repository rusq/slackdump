============
Slack Dumper
============
Purpose: dump slack messages, users and files using browser token and cookie.

This library is "fit-for-purpose" quality, can't say it's ready for
production, as it lacks most of the unit tests, but will do for ad-hoc
use.

Usage
=====

1. Download the ``slackdump`` for your operating system.
2. Unpack
3. Run ``slackdump -h`` to see help.

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
   executable in any text editor.
#. Add the token and cookie values to it. End result
   should look like this::

     SLACK_TOKEN=xoxc-<...elided...>
     COOKIE=12345472908twp<...elided...>

#. Save the file and close the editor.


Dumping conversations
~~~~~~~~~~~~~~~~~~~~~

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

Say, you want to dump convesations with @alice and @bob to the text
files and also want to save all the files that you all shared in those
convesations::

  slackdump -r text -f DNF3XXXXX DLY4XXXXX
       	    ------- -- --------- ---------
               |     |    |         |
               |     |    |         +-: @alice
               |     |    +-----------: @bob
               |     +----------------: save files
               +----------------------: text file output

Dumping users
~~~~~~~~~~~~~

To view all users, run::

  slackdump -u

	       
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


.. _Application: https://stackoverflow.com/questions/12908881/how-to-copy-cookies-in-google-chrome
