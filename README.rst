============
Slack Dumper
============

- `Buy me a cup of tea`_
- Join the discussion in Telegram_ or Slack_.
- `Read the overview on Medium.com`_
- .. image:: https://pkg.go.dev/badge/github.com/rusq/slackdump/v2.svg
     :alt: Go Reference
     :target: https://pkg.go.dev/github.com/rusq/slackdump/v2/

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

There a three modes of operation:

- List users/channels
- Dumping messages and threads
- Creating a Slack Export.

Slackdump accepts two types of input:

#. the URL/link of the channel or thread, OR 
#. the ID of the channel.

.. contents::
   :depth: 2

Users Manual
============

`Users Manual`_ is located in the doc_ directory.


Using as a library
==================

Download:

.. code:: go

  go get github.com/rusq/slackdump/v2

Use:

.. code:: go

  import "github.com/rusq/slackdump/v2"

  func main() {
    sd, err := slackdump.New(os.Getenv("TOKEN"), os.Getenv("COOKIE"))
    if err != nil {
        // handle
    }
    // ... read the docs
  }

FAQ
===

:Q: **Do I need to create a Slack application?**

:A: No, you don't.  You need to grab that token and cookie from the
    browser Slack session.  See Usage_ at the top of the file.

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
.. _Users Manual: doc/README.rst


..
  bulletin board links

.. _`TheSignChef.com`: https://www.glassdoor.com.au/Reviews/TheSignChef-com-Reviews-E793259.htm
.. _`Get cookies.txt Chrome extension`: https://chrome.google.com/webstore/detail/get-cookiestxt/bgaddhkoddajcdgocldbbfleckgcbcid
