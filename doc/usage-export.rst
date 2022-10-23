Creating Slack Export
---------------------
[Index_]

.. contents::

This feature allows one to create a slack export of the Slack workspace in
standard or Mattermost compatible format.

There are four main export specific command line flags that control the export
behaviour:

-export string
  Enables the export mode and allows to specify the file or directory to save
  the data to, for example::
    
    -export my_export.zip

-export-type string (optional)
  Allows to specify the export type.  It mainly affects how the location of
  attachments files within the archive.  It can accept the following values::
    
    standard    - attachments are placed into channel_id/attachments directory.
    mattermost  - attachments are placed into __uploads/ directory

  ``standard`` is the default export mode, if this parameter is not specified.

Example::
    
    slackdump -export my_export.zip -export-type mattermost


-export-token string (optional)
  Allows to append a custom export token to all attachment files (even if the
  download is disabled).  It modifies each file's Download URLs and Thumbnail
  URLs by adding the t= URL value to them.  NOTE: if you don't want it to be
  saved in shell history, specify it as an environment variable
  "SLACK_FILE_TOKEN", i.e.::

    SLACK_FILE_TOKEN=xoxe-.... slackdump -export my_export.zip

-download (optional)
  If this flag is present, Slackdump will download attachments::

    slackdump -export my_export.zip -download


Export Types
~~~~~~~~~~~~

By default, Slackdump generates the Standard type Export. 

The export file or directory will include emails and, if
``-download`` flag is specified, attachments.

Mattermost Export
+++++++++++++++++

Mattermost mode is currently in alpha-stage.  Export is generated in the
format that can be imported using Mattermost "bulk" import mode format using
``mmetl/mmctl`` tools (see quick guide below).

The ``mattermost import slack`` command is not yet supported.

Mattermost Export Quick Guide
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

To export to Mattermost, Slackdump should be started with ``-export-type
mattermost`` flag.  Mattermost tools would require a ZIP file.

Steps to export from Slack and import to Mattermost:

#. Run Slackdump in mattermost mode to export the workspace::

     slackdump -export my-workspace.zip -export-type mattermost -download

   optionally, you can specify list of conversation to export::

     slackdump -export my-workspace.zip -export-type mattermost -download C12301120 D4012041

#. Download the ``mmetl`` tool for your architecture from `mmetl
   github page`_.  In the example we'll be using the Linux version::

     curl -LO https://github.com/mattermost/mmetl/releases/download/0.0.1/mmetl.linux-amd64.tar.gz

   Unpack::

     tar zxf mmetl.linux-amd64.tar.gz

#. Run the ``mmetl`` tool to generate the mattermost bulk import
   JSONL file::

     ./mmetl transform slack -t Your_Team_Name -d bulk-export-attachments -f test.zip -o mattermost_import.jsonl

   For example, if your Mattermost team is "slackdump"::

     ./mmetl transform slack -t slackdump -d bulk-export-attachments -f test.zip -o mattermost_import.jsonl
     
   This will generate a directory ``bulk-export-attachments`` and
   ``mattermost_import.jsonl`` file in the current directory.

#. Create a zip archive in bulk format.  Please ensure that the
   ``bulk-export-attachments`` directory is placed inside ``data``
   directory by following the steps below::

     mkdir data
     mv bulk-export-attachments data
     zip -r bulk_import.zip data mattermost_import.jsonl

#. Copy the resulting file to the mattermost server, and upload it using ``mmctl`` tool::

     mmctl import upload ./bulk_import.zip

   This will upload the zip file into the Mattermost.

   **NOTE**: you may need to authenticate to use ``mmctl``. Run::

     mmctl auth login URL
     # URL is the URL of your mattermost server, i.e.:
     mmctl auth login http://localhost:8065

   List all import files to find out the filename that will be used to
   start the import process::

     mmctl import list available

   The output will print the file with an ID prefix::
     
     9zgyay5wupdyzc1kqdin5re77e_bulk_import.zip

#. Start the import process::

     mmctl import process <filename>

   For example::

     mmctl import process 9zgyay5wupdyzc1kqdin5re77e_bulk_import.zip
     
#. To monitor the status of the job or to see if there are any
   errors::

     mmctl import job list

   and::

     mmctl import job show <JOB ID> --json

After following all these steps, you should see the data in your
Mattermost team.
     
More detailed instructions can be found in the `Mattermost
documentation`_

Mattermost Export Directory Structure
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

The Mattermost type archive will have the following structure::

  /
  ├── __uploads              : all uploaded files are placed in this dir.
  │   └── F02PM6A1AUA        : slack file ID is used as a directory name
  |       └── Chevy.jpg      : file attachment
  ├── everyone               : channel "#everyone"
  │   ├── 2022-01-01.json    :   all messages for the 1 Jan 2022.
  │   └── 2022-01-04.json    :    "     "      "   "  4 Jan 2022.
  ├── DM12345678             : Your DMs with Scumbag Steve^
  │   └── 2022-01-04.json    :   (you did not have much to discuss —
  │                          :    Steve turned out to be a scumbag)
  ├── channels.json          : all workspace channels information
  ├── dms.json               : direct message information
  └── users.json             : all workspace users information

Standard Export
+++++++++++++++

To run in Slack Export standard mode, one must start Slackdump
specifying the slack export directory or zip file, i.e.::

  slackdump -export my-workspace -export-type standard

  < OR, for a ZIP file >

  slackdump -export my-workspace.zip -export-type standard

Slackdump will export the whole workspace.  If ' ``-download``' flag is
specified, all files will be saved under the channel's '``attachments``'
directory.

Standard Export Directory Structure
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

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

Inclusive and Exclusive Export
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

It is possible to **include** or **exclude** channels in/from the Export.

Exporting Only Channels You Need
++++++++++++++++++++++++++++++++

To **include** only those channels you're interested in, use the following
syntax::

  slackdump -export my-workspace.zip C12401724 https://xxx.slack.com/archives/C4812934

The command above will export ONLY channels ``C12401724`` and ``C4812934``.

Exporting Everything Except Some Unwanted Channels
++++++++++++++++++++++++++++++++++++++++++++++++++

To **exclude** one or more channels from the export, prefix the channel with "^"
character.  For example, you want to export everything except channel C123456::

  slackdump -export my-workspace.zip ^C123456

Providing the List in a File
++++++++++++++++++++++++++++

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

  Slack Export is currently in beta development stage, please open an
  issue_ in Github Issues, if you run into problems.


Migrating to
~~~~~~~~~~~~

Discord
+++++++

The preferred way is to use Slackord2_ - a great tool with a nice GUI that is
compatible with Slackdump generated export files.  If you have any
compatibility issues, please open a Github issue_.

Viewing export
~~~~~~~~~~~~~~

SlackLogViewer
++++++++++++++

SlackLogViewer_ is a fast desktop application, with an advanced search
function that turns your Slack Export file into a searchable knowledge base.
It is extremely fast due to being written in C++ and comes as a single
executable.  Recently it was updated to support the preview of DMs.

`Download SlackLogViewer`_ v1.2.

Slack Export Viewer
+++++++++++++++++++

While you're welcome to just open each individual ``.json`` file to read the
contents of your backup, you might also consider using a tool like
`slack-export-viewer <https://github.com/hfaran/slack-export-viewer>`_. Some
work has been put in, to make ``slackdump`` compatible with
``slack-export-viewer``, which will allow you to navigate your backup with a
slack-like GUI.

[Index_]

.. _`Scumbag Steve`: https://www.google.com/search?q=Scumbag+Steve
.. _Index: README.rst
.. _mmetl github page: https://github.com/mattermost/mmetl
.. _Mattermost documentation: https://docs.mattermost.com/onboard/migrating-to-mattermost.html#migrating-from-slack-using-the-mattermost-mmetl-tool-and-bulk-import
.. _Slackord2: https://github.com/thomasloupe/Slackord2
.. _issue: https://github.com/rusq/slackdump/issues
.. _SlackLogViewer: https://github.com/thayakawa-gh/SlackLogViewer
.. _Download SlackLogViewer: https://github.com/thayakawa-gh/SlackLogViewer/releases
