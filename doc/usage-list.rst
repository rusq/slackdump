=========================
Dumping Users or Channels
=========================
[Index_]

.. contents::

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
output json, use '``-r json``' flag.

[Index_]

.. _Index: README.rst