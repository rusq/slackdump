============
Slack Dumper
============

Purpose:  archive your private and public Slack messages, users, channels,
files and emojis.  Generate Slack Export without admin privileges.

|screenshot|

**Quick links**:

- Join the discussion in Telegram_.
- `Buy me a cup of tea`_, or use **Github Sponsors** button on the top of the
  page.
- Reference documentation: |go ref|
- How to's:

  - `Mattermost migration`_ steps
  - `SlackLogViewerとSlackdumpを一緒に使用する`_
  - `Step by Step guide by Viviana Marquez`_ (requires Medium subscription)
  - `Overview on Medium.com`_  (outdated)

.. contents::
   :depth: 2

Description
===========

Typical use scenarios:

* archive your private conversations from Slack when the administrator
  does not allow you to install applications OR you don't want to use
  potentially privacy-violating third-party tools,
* archive channels from Slack when you're on a free "no archive" subscription,
  so you don't lose valuable knowledge in those channels,
* create a Slack Export archive without admin access, or
* save your favourite emojis.

There are four modes of operation (more on this in `User Guide`_) :

#. List users/channels
#. Dumping messages and threads
#. Creating a Slack Export in Mattermost or Standard modes.
#. Emoji download mode.

Slackdump accepts two types of input (see `Dumping Conversations`_ section):

#. the URL/link of the channel or thread, OR
#. the ID of the channel.


Quick Start
===========

#. Download the latest release for your operating system from the releases_
   page. (If you're using **macOS**, download **Darwin** executable).
#. Unpack the archive to any directory.
#. Run the ``./slackdump`` or ``slackdump.exe`` executable (see note below).
#. You know the drill:  use arrow keys to select the menu item, and Enter (or
   Return) to confirm.

By default, Slackdump uses the EZ-Login 3000 automatic login, and interactive
mode.

.. NOTE::
  On Windows and macOS you may be presented with "Unknown developer" window,
  this is fine.  Reason for this is that the executable hasn't been signed by
  the developer certificate.

  To work around this:

  - **on Windows**: click "more information", and press "Run
    Anyway" button.
  - **on macOS**: open the folder in Finder, hold Option and double click the
    executable, choose Run.


Slackord2: Migrating to Discord
===============================

If you're migrating to Discord, the recommended way is to use Slackord2_ - a
great tool with a nice GUI, that is compatible with the export files generated
by Slackdump.

User Guide
==========

For more advanced features and instructions, please see the `User Guide`_.

Previewing Results
==================

Once the data is dumped, you can use one of the following tools to preview the
results:

- `SlackLogViewer`_ - a fast and powerful Slack Export viewer written in C++.
- `Slackdump2Html`_ - a great Python application that converts Slack Dump to a
  static browsable HTML, works on Dump mode files.
- `slack export viewer`_ - Slack Export Viewer is a well known viewer for
  slack export files.

Using as a library
==================

Download:

.. code:: go

  go get github.com/rusq/slackdump/v2


Example
-------
.. code:: go

  package main

  import (
    "context"
    "log"

    "github.com/rusq/slackdump/v2"
    "github.com/rusq/slackdump/v2/auth"
  )

  func main() {
    provider, err := auth.NewValueAuth("xoxc-...", "xoxd-...")
    if err != nil {
        log.Print(err)
        return
    }
    sd, err := slackdump.New(context.Background(), provider)
    if err != nil {
        log.Print(err)
        return
    }
    _ = sd
  }

See |go ref|

Using Custom Logger
-------------------
Slackdump uses a simple `rusq/dlog`_ as a default logger (it is a wrapper around
the standard logger that adds `Debug*` functions).

If you want to use the same default logger that Slackdump uses in your
application, it is available as ``logger.Default``.

No doubts that everyone has their own favourite logger that is better than other
miserable loggers.  Please read below for instructions on plugging your
favourite logger.


Logrus
~~~~~~
Good news is logrus_ can be plugged in straight away, as it implements the
``logger.Interface`` out of the box.

.. code:: go

  lg := logrus.New()
  sd, err := slackdump.New(context.Background(), provider, WithLogger(lg))
    if err != nil {
        log.Print(err)
        return
    }
  }


Glog and others
~~~~~~~~~~~~~~~
If you need to use some other logger, such as glog_, it is a matter of wrapping
the calls to satisfy the ``logger.Interface`` (defined in the `logger`_
package), and then setting the ``Logger`` variable in `slackdump.Options` (see
`options.go`_), or using `WithLogger` option.


FAQ
===

:Q: **Do I need to create a Slack application?**

:A: No, you don't.  Just run the application and EZ-Login 3000 will take
    care of the authentication or, alternatively, grab that token and
    cookie from the browser Slack session.  See `User Guide`_.

:Q: **I'm getting "invalid_auth" error**

:A: Go get the new Cookie from the browser and Token as well.

:Q: **Slackdump takes a very long time to cache users**

:A: Disable the user cache with ``-no-user-cache`` flag.

:Q: **How to read the export file?**

:A: For Slack Workspace Export, use SlackLogViewer_ which is extremely fast
    with an advanced search function, or `slack export viewer`_ which is a
    Python application and runs in a browser.  For the generic dump files, see
    `examples`_ directory for some python and shell examples.

:Q: **My Slack Workspace is on the Free plan.  Can I get data older than
    90-days?**

:A: No, unfortunately you can't.  Slack doesn't allow to export data older
    than 90 days for free workspaces, the API does not return any data before 90
    days for workspaces on the Free plan.

Thank you
=========
Big thanks to all contributors, who submitted a pull request, reported a bug,
suggested a feature, helped to reproduce, or spent time chatting with me on
the Telegram or Slack to help to understand the issue and tested the proposed
solution.

Also, I'd like to thank all those who made a donation to support the project:

- Vivek R.
- Fabian I.
- Ori P.
- Shir B. L.
- Emin G.
- Robert Z.
- Sudhanshu J.

Bulletin Board
--------------

Messages that were conveyed with the donations:

- 25/01/2022: Stay away from `TheSignChef.com`_, ya hear, they don't pay what
  they owe to their employees.


.. _`Buy me a cup of tea`: https://ko-fi.com/rusq_
.. _Telegram: https://t.me/slackdump
.. _`Overview on Medium.com`: https://medium.com/@gilyazov/downloading-your-private-slack-conversations-52e50428b3c2
.. _User Guide: doc/README.rst
.. _Dumping Conversations: doc/usage-channels.rst
.. _Mattermost migration: doc/usage-export.rst
.. _rusq/dlog: https://github.com/rusq/dlog
.. _logrus: https://github.com/sirupsen/logrus
.. _glog: https://github.com/golang/glog
.. _logger: logger/logger.go
.. _options.go: options.go
.. _examples: examples
.. _slack export viewer: https://github.com/hfaran/slack-export-viewer
.. _releases: https://github.com/rusq/slackdump/releases/
.. _Slackord2: https://github.com/thomasloupe/Slackord2
.. _SlackLogViewer: https://github.com/thayakawa-gh/SlackLogViewer/releases
.. _Slackdump2Html: https://github.com/kununu/slackdump2html
.. _`Step by Step guide by Viviana Marquez`: https://vivianamarquez.medium.com/a-step-by-step-guide-to-downloading-slack-messages-without-admin-rights-954f20397e83
.. _`SlackLogViewerとSlackdumpを一緒に使用する`: https://kenkyu-note.hatenablog.com/entry/2022/09/02/090949

..
  bulletin board links

.. _`TheSignChef.com`: https://www.glassdoor.com.au/Reviews/TheSignChef-com-Reviews-E793259.htm

.. |go ref| image:: https://pkg.go.dev/badge/github.com/rusq/slackdump/v2.svg
              :alt: Go Reference
           :target: https://pkg.go.dev/github.com/rusq/slackdump/v2/

.. |screenshot| image:: doc/slackdump.webp
               :target: https://github.com/rusq/slackdump/releases/
