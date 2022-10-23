=======================
 Manual Authentication
=======================
[Index_]

Getting the authentication data
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

#. Open up your Slack *in your browser* and login.

TOKEN
+++++

#. Open your browser's *Developer Console*.

   #. In Firefox, under `Tools -> Browser Tools -> Web Developer tools` in the menu bar
   #. In Chrome, click the 'three dots' button to the right of the URL Bar, then select
      'More Tools -> Developer Tools'
#. Switch to the console tab.
#. Paste the following snippet and press ENTER to execute::

     JSON.parse(localStorage.localConfig_v2).teams[document.location.pathname.match(/^\/client\/(T[A-Z0-9]+)/)[1]].token

#. Token value is printed right after the executed command (it starts with
   "``xoxc-``"), save it somewhere for now.

.. NOTE:: if you're having problems running the code snippet above, you can
          get the token the conventional way, see Troubleshooting_ section below.

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

   Alternatively, if you saved the cookies to the file, it will look like this::

     SLACK_TOKEN=xoxc-<...elided...>
     COOKIE=path/to/slack.com_cookies.txt

#. Save the file and close the editor.

Troubleshooting
~~~~~~~~~~~~~~~

Getting token the hard way
++++++++++++++++++++++++++

#. Open your browser's *Developer Console*, as described in the TOKEN_ section
   steps above.
#. Go to the Network tab
#. In the toolbar, switch to ``Fetch/XHR`` view.
#. Open any channel or private conversation in Slack.  You'll see a
   bunch of stuff appearing in Network panel.
#. In the list of requests, find the one starting with
   ``channels.prefs.get?``, click it and click on *Headers* tab in the
   opened pane.
#. Scroll down, until you see **Form Data**
#. Grab the **token:** value (it starts with "``xoxc-``"), by right
   clicking the value and choosing "Copy Value".

**If you don't see the token value** in Google Chrome - switch to `Payload` tab,
your token is waiting for you there.


[Index_]

.. _Index: README.rst
.. _Application: https://stackoverflow.com/questions/12908881/how-to-copy-cookies-in-google-chrome
.. _`Get cookies.txt Chrome extension`: https://chrome.google.com/webstore/detail/get-cookiestxt/bgaddhkoddajcdgocldbbfleckgcbcid
