============
Slack Dumper
============

- `Buy me a cup of tea`_
- Join the discussion in Telegram_ or Slack_.
- `Read the overview on Medium.com`_
- |go ref|


.. contents::
   :depth: 2

Description
===========

Purpose: dump Slack messages, users and files using browser token and cookie.

Typical use scenarios:

* archive your private conversations from Slack when the administrator
  does not allow you to install applications OR you don't want to use 
  potentially privacy-violating third-party tools, 
* archive channels from Slack when you're on a free "no archive" subscription,
  so you don't lose valuable knowledge in those channels.
* create a Slack Export archive without admin access.

There a three modes of operation (more on this in `User Guide`_) :

#. List users/channels
#. Dumping messages and threads
#. Creating a Slack Export.

Slackdump accepts two types of input (see `Dumping Conversations`_ section):

#. the URL/link of the channel or thread, OR 
#. the ID of the channel.


Quick Start
===========

Please see the `User Guide`_.


Using as a library
==================

Download:

.. code:: go

  go get github.com/rusq/slackdump/v2

Add the following line at the end of your project's ``go.mod`` file::

  replace github.com/slack-go/slack => github.com/rusq/slack v0.11.100

This is required, as Slackdump relies on custom autorization scheme
that uses cookies, and those functions are simply not in the original
library.

Use:

.. code:: go

  import (
    "github.com/rusq/slackdump/v2"
    "github.com/rusq/slackdump/v2/auth"
  )

  func main() {
    provider, err := auth.NewValueAuth("xoxc-...", "xoxd-...")
    if err != nil {
        log.Print(err)
        return
    }
    sd, err := New(context.Background(), provider)
    if err != nil {
        log.Print(err)
        return
    }
    _ = sd
  }

See |go ref|

FAQ
===

:Q: **Do I need to create a Slack application?**

:A: No, you don't.  Just run the application and EZ-Login 3000 will take
    care of the authentication or, alternatively, grab that token and
    cookie from the browser Slack session.  See `User Guide`_.

:Q: **I'm getting "invalid_auth" error**

:A: Go get the new Cookie from the browser and Token as well.


Bulletin Board
--------------

Messages that were conveyed with the donations:

- 25/01/2022: Stay away from `TheSignChef.com`_, ya hear, they don't pay what
  they owe to their employees. 

.. _Application: https://stackoverflow.com/questions/12908881/how-to-copy-cookies-in-google-chrome
.. _`Buy me a cup of tea`: https://www.paypal.com/donate/?hosted_button_id=GUHCLSM7E54ZW
.. _Telegram: https://t.me/slackdump
.. _Slack: https://join.slack.com/t/newworkspace-wcx3986/shared_invite/zt-18kj2sdoj-jMi3aZMWwkbK5JNjne0dbQ
.. _`Read the overview on Medium.com`: https://medium.com/@gilyazov/downloading-your-private-slack-conversations-52e50428b3c2
.. _`Go templating`: https://pkg.go.dev/html/template
.. _User Guide: doc/README.rst
.. _Dumping Conversations: doc/usage-channels.rst

..
  bulletin board links

.. _`TheSignChef.com`: https://www.glassdoor.com.au/Reviews/TheSignChef-com-Reviews-E793259.htm
.. _`Get cookies.txt Chrome extension`: https://chrome.google.com/webstore/detail/get-cookiestxt/bgaddhkoddajcdgocldbbfleckgcbcid

.. |go ref| image:: https://pkg.go.dev/badge/github.com/rusq/slackdump/v2.svg
              :alt: Go Reference
           :target: https://pkg.go.dev/github.com/rusq/slackdump/v2/
