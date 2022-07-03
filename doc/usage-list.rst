=========================
Dumping Users or Channels
=========================
[Index_]

.. contents::

General Information
-------------------

Both Users and Channels dump modes support the following flags:

- ``-r`` - sets the output format, can be ``text`` or ``json``.  Default is
  ``text``.
- ``-o`` - optional flag to set the output filename.  If output filename is not
  specified, the Users or Channels will be printed on the screen.

Dumping users
-------------

To view all users, run::

  slackdump -list-users

or::

  slackdump -u


If the channel list in your Slack Workspace is too large, you can skip the
caching of users by specifying the ``-no-user-cache`` flag::

  slackdump -no-user-cache

In this case, the users will not be cached.  This flag works with both `generic
dump`_ and `Slack Export`_ modes.

Viewing Conversations
---------------------

To view all Conversations, that are visible to your account, including group
conversations, archived chats and public channels, run::

  slackdump -list-channels

or::

  slackdump -c

The output may look like this::

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

Please note, that if your Workspace contains a large amount of channels, the
channel listing will take a long time to run.

The Slack Worskpace of 20,000 channels takes around 1 hour to retrieve the
channel information from Slack.  Why?  Because Slack rate limits are tough, and
even adhering to those limits may get you rate limited.

[Index_]

.. _Index: README.rst
.. _generic dump: usage-channels.rst
.. _Slack Export: usage-export.rst
