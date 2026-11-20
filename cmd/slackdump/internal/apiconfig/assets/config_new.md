# "New" Command

Creates a new API configuration file containing default values. You will need
to specify the filename, for example:

    slackdump config new myconfig.toml

If the extension is omitted, ".toml" is automatically appended to the name of
the file.

Configuration file contains the following groups of settings:
- File download concurrency and retries;
- Rate limits;
- Batch sizes per request;

### Slack Rate Limits
Slack imposes rate limits on API calls. The default values are set to the
maximum allowed by Slack. If you want to change the rate limits, you can do so
in the configuration file.

Slack API has four tiers of [rate limits][1], with Tier 1 being the most
restrictive and Tier 4 being the least restrictive. The rate limits are
measured in requests per minute and enforced on a (token,method) pair.

Let's look at the example configuration file:

```toml
{{ .DEFAULT_LIMITS_TOML -}}
```

The base Tier values are hardcoded in the application, but the configuration
file allows to tweak the "boost" and "burst" values for each tier.

The "boost" value is the number of requests that slackdump will make *on top*
of the base rate limit. **For example**: "Slack Web API Tier 2" has base limit
of 20+ requests per minute, but if "boost" is set to 20, Slackdump will make 40
requests per minute.

The "burst" value is the maximum number of requests Slackdump's limiter can
allow immediately when its token bucket is full. It is passed directly as an
argument to the [rate limiting library][2].

"Retries" is the maximum number of attempts, including the initial request.
A value of 0 uses three attempts. Slackdump retries the following errors:
- Rate limit exceeded;
- EOF or unexpected EOF.
- Network read or write errors.
- HTTP 408 and 500–599 except 501.
- For read-only calls that opt in, Slack API errors `invalid_auth`,
  `fatal_error`, `internal_error`, and `service_unavailable`.

If Slackdump receives a recoverable error, it will do one of the following:
- If it is a rate limit error, it will wait for the specified amount of time
  and retry the request;
- For network read or write errors, it will use the exponential backoff algorithm to
  wait and retry the request up to a limit of 5 minutes.
- For other types of recoverable errors, it will use the cubic backoff, capped
  at 5 minutes as well.


[1]: https://docs.slack.dev/apis/web-api/rate-limits/
[2]: https://pkg.go.dev/golang.org/x/time/rate#NewLimiter
