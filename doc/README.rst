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

   .. tip:: **MacOS users** should download ``darwin`` release file.
#. Unpack;
#. Change directory to where you have unpacked the archive;
#. Run ``./slackdump -h`` to view help options.

For compiling from sources see: `Compiling from sources`_

Logging in
----------
There are two types of login options available:

- Automatic_ (**EZ-Login 3000**, works only in 64-bit systems); OR
- Manual_

Automatic_ login is the default one, it requires no prior setup, and the
general recommendation is to use the Automatic login.  If the Automatic login
doesn't work for some reason, fallback to Manual_ login steps.

Usage
-----
There are four modes of operation:

- `Listing users/channels`_
- `Dumping messages and threads`_ (private and public)
- `Creating a Slack export`_
- `Downloading all Emojis`_


.. _Automatic:  login-auto.rst
.. _Manual: login-manual.rst
.. _Installation: usage-install.rst
.. _Dumping messages and threads: usage-channels.rst
.. _Creating a Slack Export: usage-export.rst
.. _Listing users/channels:  usage-list.rst
.. _Downloading all Emojis:  usage-emoji.rst
.. _Releases: https://github.com/rusq/slackdump/releases
.. _Compiling from sources: compiling.rst
.. _Unix Shell Guide: https://swcarpentry.github.io/shell-novice/
