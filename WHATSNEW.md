# What's new

## What's new in Slackdump v3.2

- New channel type filtering via `--chan-types` and wizard multi-select, wired through list/archive/export/resume flows.
- Optional custom profile field labels with `--custom-labels`, including UI support; uses a new user profile fetch path.
- Channel type constants now align with Slack string values; channel retrieval defaults to all types when none specified.
- Listing commands now report empty results early and expose list sizes; added tests for list length helpers.
- Internal stream/control updates for custom user profile fetching, plus expanded mocks and tests.
- Safer enum String() methods guard against negative values across generated stringers.

