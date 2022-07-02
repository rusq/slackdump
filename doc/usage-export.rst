Creating Slack Export
---------------------
[Index_]

.. contents::

Exporting Slack Workspace
~~~~~~~~~~~~~~~~~~~~~~~~~

This feature allows one to create a slack export of the slack workspace. To
run in Slack Export mode, one must start Slackdump specifying the
slack export directory, i.e.::

  slackdump -export my-workspace

Or, if you want to save export as a ZIP file::

  slackdump -export my-workspace.zip

Slackdump will export the whole workspace.  If ' ``-f``' flag is specified,
all files will be saved under the channel's '``attachments``' directory.

Inclusive and Exclusive export
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

It is possible to **include** or **exclude** channels in/from the Export.

Exporting only the channels you need
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

To **include** only those channels you're interested in, use the following
syntax::

  slackdump -export my-workspace.zip C12401724 https://xxx.slack.com/archives/C4812934

The command above will export ONLY channels ``C12401724`` and ``C4812934``.

Exporting everything except some unwanted channels
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

To **exclude** one or more channels from the export, prefix the channel with "^"
character.  For example, you want to export everything except channel C123456::

  slackdump -export my-workspace.zip ^C123456

Providing the list in a file
^^^^^^^^^^^^^^^^^^^^^^^^^^^^

You can specify the filename instead of listing all the channels on the command
line.  To include the channels from the file, use the "@" character prefix.  The
following example shows how to load the channels from the file named
"data.txt"::

  slackdump -export my-workspace.zip @data.txt

It is also possible to combine files and channels, i.e.::

  slackdump -export everything.zip @data.txt ^C123456

The command above will read the channels from ``data.txt`` and exclude the
channel ``C123456`` from the Export.

.. Note::

  Slack Export is currently in beta development stage, please report
  all issues in Github `Issues <https://github.com/rusq/slackdump/issues>`_.

Slack Export Directory Structure
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Sample directory or ZIP file structure::

  /
  ├── everyone               : channel "#everyone"
  │   ├── 2022-01-01.json    :   all messages for the 1 Jan 2022.
  │   ├── 2022-01-04.json    :    "     "      "   "  4 Jan 2022.
  │   └── attachments        :   message files
  │       └── F02PM6A1AUA-Chevy.jpg       : message attachment
  ├── DM12345678             : Your DMs with Scumbag Steve^
  │   └── 2022-01-04.json    :   (you did not have much to discuss —
  │                          :    Steve turned out to be a scumbag)
  ├── channels.json          : all workspace channels information
  ├── dms.json               : direct message information
  └── users.json             : all workspace users information

Channels
  The channels are be saved in directories, named after the channel title, i.e.
  ``#random`` would be saved to "random" directory.  The directory will contain
  a set of JSON files, one per each day.

Users
  User directories will have an "D" prefix, to find out the user name, check
  ``users.json`` file.

Group Messages
  Group messages will have name listing all the users handles involved.

^In case you're wondering who's `Scumbag Steve`_.

Slack Export Viewer
~~~~~~~~~~~~~~~~~~~

While you're welcome to just open each individual ``.json`` file to read the
contents of your backup, you might also consider using a tool like
`slack-export-viewer <https://github.com/hfaran/slack-export-viewer>`_. Some
work has been put in, to make ``slackdump`` compatible with
``slack-export-viewer``, which will allow you to navigate your backup with a
slack-like GUI.

[Index_]

.. _`Scumbag Steve`: https://www.google.com/search?q=Scumbag+Steve
.. _Index: README.rst
