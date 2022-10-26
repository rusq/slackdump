package man

import "github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"

var ManLogin = &base.Command{
	UsageLine: "login",
	Short:     "login related information",
	Long: `
	
# Login #

Slackdump supports the following login methods:

Automatic:
  - Browser authentication (EZ-Login 3000);

Manual:
  - Login with Client Token and Cookie;
  - Login with Client Token and Cookie file, exported from your browser;
  - Login with Legacy, Application or Bot Token (no cookie needed).


## EZ-Login 3000 ##

If the -token flag is not specified, Slackdump starts the EZ-Login 3000.  The
process of login is as follows:

  - You will be asked the Slack workspace name that you wish to login to;
  - After you have entered the workspace name, the browser will open.  This
    browser has nothing to do with the browser on your device, so there are
    no stored passwords or history;
  - Once you entered the correct username and password, the Slack login process
    begins;
  - The browser will close automatically, and the Token and Cookies are
    captured.

After this, if you have provided a command to run, it will start exectuion,
otherwise, if no commands are given, an interactive menu of Slackdump Wizard
displayed.

`,
}
