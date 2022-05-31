Compiling from Sources
----------------------
[Index_]

Slackdump uses a slightly `modified`_ "slack" library via module replacement
directive to enable the cookie authentication, so ``go install`` won't work.  To
compile it from sources, run the following commands::
 
   git clone github.com/rusq/slackdump
   cd slackdump
   go build ./cmd/slackdump

.. _Index: README.rst
.. _modified: https://github.com/rusq/slack
