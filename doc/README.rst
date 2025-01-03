===========
 Slackdump
===========

.. contents::

Beginner's Guide to Command Line
--------------------------------
If you have no experience working with the Linux/macOS Terminal or Windows
Command Prompt, please read this `Unix Shell Guide`_.

Installation
------------

Installing is pretty simple - just download the latest Slackdump from the
Releases_ page, extract and run it:

#. Download the archive from the Releases_ page for your operating system.

   .. tip:: **MacOS users** can use ``brew install slackdump`` to install the
      latest version.
#. Unpack;
#. Change directory to where you have unpacked the archive;
#. Run ``./slackdump -h`` to view help options.

For compiling from sources see: `Compiling from sources`_

Logging in
----------

See the quickstart_ guide on how to login.

If you need a token, you can use Manual_ steps, save them to the file and then
import them with::

  slackdump workspace import <filename>

Read more in Workspace Import_.

.. _quickstart: https://github.com/rusq/slackdump/blob/master/cmd/slackdump/internal/man/assets/quickstart.md
.. _Import: https://github.com/rusq/slackdump/blob/master/cmd/slackdump/internal/workspace/assets/import.md

Usage
-----
There are several modes of operation:

- `Listing users/channels`_
- `Dumping messages and threads`_ (private and public)
- `Creating a Slack export`_
- `Downloading all Emojis`_


.. _Manual: login-manual.rst
.. _Installation: usage-install.rst
.. _Dumping messages and threads: usage-channels.rst
.. _Creating a Slack Export: usage-export.rst
.. _Listing users/channels:  usage-list.rst
.. _Downloading all Emojis:  usage-emoji.rst
.. _Releases: https://github.com/rusq/slackdump/releases
.. _Compiling from sources: compiling.rst
.. _Unix Shell Guide: https://swcarpentry.github.io/shell-novice/
