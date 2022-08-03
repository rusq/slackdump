==============================
Automatic login: EZ-Login 3000
==============================
[Index_]

|ez-login|

*EZ-Login 3000* provides an easy solution to login to Slack without
the complex steps of manually extracting the token and cookies from
the browser.  To use it - just start the Slackdump, and you will be
presented with *EZ-Login 3000* Welcome Screen.  Even scientists can't
explain why it feels so warm and welcoming.

Quickstart
==========

When you start the Slackdump, the EZ-Login 3000 prompt will appear
along with some instructions, asking for the Slack Workspace Name.
You can type '?' and press Enter to get instructions::

  ? Enter Slack Workspace Name:  [? for help] _

You can paste the URL of your Slack Workspace, or just type in the
name of the workspace.  For example::

  ? Enter Slack Workspace Name:  [? for help] https://evilcorp.slack.com

OR::

  ? Enter Slack Workspace Name:  [? for help] evilcorp

To get help, type '?' and press [Enter].  Once you have entered the
Slack workspace name, press [Enter] (or [Return] on some keyboards), a
browser window will appear, and the given Slack Workspace Login page
will open.

Login the usual way for your workspace (i.e. by entering your email
and password, or using the Single-Sign-On).

When you press the login button, the Slack will start loading the
workspace and then Browser will close automatically and the Slackdump
will be logged in.

Your credentials will be stored in an encrypted file in a Slackdump
cache subdirectory of your user's Local Cache directory.

How safe is the storage
=======================

The encrypted file is protected with AES-256 encryption using your
Operating System unique Machine ID.  It is not possible to copy it to
another machine:  it will not be able to be decrypted anywhere else
but your computer.

Reset the Authentication (Logout)
=================================

In order to be able to login again (or to another workspace), run the
Slackdump with ``-auth-reset`` flag::

  ./slackdump -auth-reset

This will delete the stored credentials and you'll be able to login
with EZ-Login 3000 again.


How exactly does this work
==========================

EZ-Login 3000 uses the playwright_ framework library to control the
browser instance.  When Slack is authenticated, EZ-Login 3000 waits
for a particular API call, and once it detects that call, it grabs the
token value and session cookies automatically to initialise the
Slackdump Client.

Troubleshooting
===============

The browser disappeared, but Slackdump doesn't do anything.
  Press [Ctrl]+[C] on your keyboard to exit Slackdump and retry
  the login procedure again.  If nothing helps, use the Manual_ login
  method.

EZ-Login 3000 prompt does not appear, and I'm getting ``invalid_auth``.
  Delete ``.env`` or ``.env.txt`` or ``secrets.txt`` files that might
  contain obsolete login credentials, and EZ-Login 3000 prompt will
  appear.

How can I reset the authentication to get the login prompt again?
  See "Reset the Authentication" above.
  
[Index_]

.. _playwright: https://playwright.dev
.. _Index: README.rst
.. _Manual: login-manual.rst

.. |ez-login| image:: ez-login.png
              :width: 320px
