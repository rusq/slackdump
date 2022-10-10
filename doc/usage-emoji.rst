==============
Dumping Emojis
==============
[Index_]

.. contents::

Slackdump allows to dump all workspace emojis.

Emoji mode requires only one parameter:

- base directory or ZIP file (``-base``)

Optional parameters:

- fail fast on errors (``-emoji-failfast``).  When download starts, the emojis
  are being downloaded using twelve goroutines.  By default, all download
  errors are printed on screen and skipped.  Specifying this flag will terminate
  the process on any download error, i.e. network failure or HTTP 404.

GUI Usage
---------

#. Start the slackdump::

    slackdump.exe
   or::

    ./slackdump
   depending on your OS;

#. choose "``Emojis``" from the main menu;
#. input the output directory or ZIP file name;
#. wait for download process to complete.

CLI Usage
---------

On windows::

  slackdump.exe -emoji -base <directory or zip file>

On *nix (including macOS)::

  ./slackdump -emoji -base <directory or zip file>

Usage Examples
~~~~~~~~~~~~~~

Create a ``my_emojis.zip`` ZIP archive on linux::

  ./slackdump -emoji -base my_emojis.zip

Create an ``emoji_dir`` on Windows::

  slackdump.exe -emoji -base emoji_dir

Output structure
----------------

The directory, or ZIP file, created by Slackdump, will have the following
structure:

- index.json: contains the index of all emojis, as returned by API.
- emojis directory: contains all emojis, that have emoji's name and png
  extension.

Please note that aliases are skipped and only original emoji will be present.
Use the ``index.json`` file to find the original name of an aliased emoji.

Output Example
~~~~~~~~~~~~~~

For example, if your workspace have the following emojis:

- \:foo:
- \:bar:
- \:baz:
- and a :foobar: alias that references :foo:
- some other emojis

The directory or ZIP file will look like this::

  .
  +- emojis
  |  +- foo.png
  |  +- bar.png
  :  :
  |  +- baz.png
  +- index.json

Search the ``index.json`` file for ``foobar``, and find out that the URL value
contains ``alias:foo``.

[Index_]

.. _Index: README.rst
