---
name: pre-release
description: Prepare this repository for a release. Use when asked to do pre-release checks, summarize changes since a previous tag, update WHATSNEW.md or changelog entries, update CONTRIBUTORS.md, audit command long help or docs for release-visible features, or verify release documentation consistency.
---

# Pre-release

Use this skill for repository-local release-prep documentation passes. Ground
every update in commit history, diffs, and current command behavior.

## Workflow

1. Determine the release range.
   - Use the base tag or version supplied by the user.
   - If no base is supplied, inspect tags with
     `git tag --list 'v*' --sort=-version:refname` and infer the previous
     release tag.
   - Inspect changes with `git log --oneline <base>..HEAD`,
     `git diff --stat <base>..HEAD`, and
     `git diff --name-only <base>..HEAD`.

2. Update release notes when requested.
   - Check whether the changelog file is a symlink before editing it.
   - Preserve the existing changelog style and section order.
   - Add the new version section above the previous release.
   - Focus on user-visible features, behavior changes, bug fixes, migration
     notes, and documented workflows.
   - Mention commands, flags, formats, and caveats exactly as implemented.
   - Avoid internal-only refactors unless they materially affect users,
     maintainers, packaging, or contributors.

3. Update contributors when requested or as part of a full release prep.
   - Compare `git shortlog -sne <base>..HEAD` and
     `git log --format='%aN <%aE>' <base>..HEAD` against `CONTRIBUTORS.md`.
   - Inspect individual commits and merge commits for contribution scope.
   - Infer GitHub handles only from reliable local evidence such as PR branch
     names in merge commits or existing repository metadata.
   - Add concise entries for new contributors and update the displayed total.
   - Do not duplicate existing contributors; extend an existing entry when
     that is clearer.

4. Audit command help and docs for release-visible changes.
   - Search embedded help assets, command definitions, and docs for each
     release-note topic.
   - Update only the help/docs that a user would naturally consult for the
     changed command, flag, format, or workflow.
   - For Go files with inline long help, avoid Markdown backticks inside raw
     string literals unless the literal delimiter allows them.
   - Keep documentation concise and practical: what changed, how to use it,
     and any important tradeoff.

5. Validate.
   - Run `gofmt` on edited Go files.
   - Run targeted tests for touched Go packages when Go files changed.
   - For Markdown-only changes, no full test run is required; say that
     explicitly.
   - Review `git diff --stat`, relevant `git diff`, and `git status --short`.
   - Preserve the user's staged and unstaged state; do not stage files unless
     asked.

## Search Hints

- Changelog: `WHATSNEW.md`, `cmd/**/assets/*changelog*`.
- Contributors: `CONTRIBUTORS.md`, `git shortlog`, merge commit messages.
- Embedded command help: `cmd/**/assets/*.md` and `Long:` fields in Go files.
- User docs: `doc/`, `README.md`, command-specific usage files.

## Final Response

Report:

- The release range inspected.
- Files changed.
- Validation commands run and results.
- Any notable staged vs unstaged state.
