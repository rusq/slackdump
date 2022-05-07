Creating Slack Export
---------------------
[Index_]

.. contents::

Exporting Slack Workspace
~~~~~~~~~~~~~~~~~~~~~~~~~

This feature allows one to create a slack export of the slack workspace. To
run in Slack Export mode, one must start Slackdump specifying the
slack export directory, i.e.::

  slackdump -export-dir my-workspace

Slackdump will export the whole workspace.  If ' ``-f``' flag is specified,
all files will be saved under the channels' '``attachments``' directory.

Slack Export is currently in alpha development stage, please report
all issues in Github Issues_.

Slack Export Directory Structure
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Sample directory structure::

  /
  ├── everyone               : channel "#everyone"
  │   ├── 2022-01-01.json    :   all messages for the 1 Jan 2022.
  │   ├── 2022-01-04.json    :    "     "      "   "  4 Jan 2022.
  │   └── attachments        :   message files
  │       └── F02PM6A1AUA-Chevy.jpg       : message attachment
  ├── IM-scumbag.steve       : Your DMs with Scumbag Steve^
  │   └── 2022-01-04.json    :   (you did not have much to discuss —
  │                          :    Steve turned out to be a scumbag)
  ├── channels.json          : all workspace channels information
  └── users.json             : all workspace users information

Channels
  The channels are be saved in directories, named after the channel title, i.e.
  ``#random`` would be saved to "random" directory.  The directory will contain
  a set of JSON files, one per each day.

Users
  User directories will have an "IM-" prefix, following by the users' Slack
  handle.

Group Messages
  Group messages will have name listing all the users handles involved.

^In case you're wondering who's `Scumbag Steve`_.

[Index_]

.. _`Scumbag Steve`: https://www.google.com/search?q=Scumbag+Steve
.. _Index: README.rst
