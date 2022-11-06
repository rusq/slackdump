# Login #

Slackdump supports the following login methods:

Automatic:
- Browser authentication (**_EZ-Login 3000_**);

Manual:
- Login with Client Token and a cookie value;
- Login with Client Token and a Cookie file, exported from your browser;
- Login with Legacy `xoxp-`, Application `xoxa-` or Bot `xoxb-` Token 
  (no cookie needed).


## EZ-Login 3000 ##

If the `-token` flag is not specified, Slackdump starts the **_EZ-Login 3000_**.
The process of login is as follows:

1. You will be asked the **Slack workspace** name that you wish to log in to;
2. After you have entered the workspace name, the browser will open.  This
   browser has nothing to do with the browser on your device, so there are
   no stored passwords or history;
3. Once you entered the correct **username** and **password**, the Slack login
   process begins;
4. The browser will close automatically, after the Token and Cookies are
   captured.

After this, if you have provided a command to run, it will start execution.
Otherwise, if no command is given, an interactive menu of **Slackdump Wizard**
is displayed.
