Creating Slack Export
---------------------
[Index_]

.. contents::

This feature allows one to create a slack export of the Slack workspace in
standard or Mattermost compatible format.

To run the export:

- In GUI mode, choose "Export" from the main menu.

- In CLI mode, run the following command::

    slackdump export


Example::
    
    slackdump -export my_export.zip -export-type mattermost

Optional arguments:

-export-token string (optional)
  Allows to append a custom export token to all attachment files (even if the
  download is disabled).  It modifies each file's Download URLs and Thumbnail
  URLs by adding the t= URL value to them.  NOTE: if you don't want it to be
  saved in shell history, specify it as an environment variable
  "SLACK_FILE_TOKEN", i.e.::

    SLACK_FILE_TOKEN=xoxe-.... slackdump -export my_export.zip


Read more by running: ``slackdump help export`` or read online_.

.. _online: https://github.com/rusq/slackdump/blob/master/cmd/slackdump/internal/export/assets/export.md


Export Types
~~~~~~~~~~~~

By default, Slackdump generates the Mattermost type Export. 

The export file or directory will include emails and, if
``-files=true`` flag is specified, attachments.

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

     slackdump export -o my-workspace.zip

   optionally, you can specify list of conversation to export::

     slackdump export my-workspace.zip C12301120 D4012041

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

Migrating to
~~~~~~~~~~~~

Discord
+++++++

The preferred way is to use Slackord2_ - a great tool with a nice GUI that is
compatible with Slackdump generated export files.  If you have any
compatibility issues, please open a GitHub issue_.

Viewing export
~~~~~~~~~~~~~~

Slackdump has a native viewer - ::

   slackdump view <export_file>

Alternatively you can use the following tools, listed below.

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

.. _Index: README.rst
.. _mmetl github page: https://github.com/mattermost/mmetl
.. _Mattermost documentation: https://docs.mattermost.com/onboard/migrating-to-mattermost.html#migrating-from-slack-using-the-mattermost-mmetl-tool-and-bulk-import
.. _Slackord2: https://github.com/thomasloupe/Slackord2
.. _SlackLogViewer: https://github.com/thayakawa-gh/SlackLogViewer
.. _Download SlackLogViewer: https://github.com/thayakawa-gh/SlackLogViewer/releases
