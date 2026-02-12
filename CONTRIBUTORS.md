# Contributors

This file recognizes all the amazing people who have contributed to Slackdump. Thank you for making this project better!

## Main Contributors

### [@rusq](https://github.com/rusq) - Rustam Gilyazov
Core maintainer and primary developer of Slackdump, responsible for the v3 rewrite, authentication system, export formats, CLI architecture, and the vast majority of features and bug fixes.

### [@yarikoptic](https://github.com/yarikoptic) - Yaroslav Halchenko
Added comprehensive code quality tools including codespell configuration and GitHub actions, improved shell scripts with shellcheck fixes, and enhanced overall code consistency.

### [@marcuscreo](https://github.com/marcus-crane) - Marcus Crane
Significantly improved the built-in viewer with static file serving, text wrapping on system messages, profile sidebar styling, file attachment improvements, and video/image display enhancements.

### [@arran4](https://github.com/arran4) - Arran Ubels
Enhanced the release pipeline with GoReleaser improvements, GitHub Actions version bumps, multiple output formats, and man page inclusion in releases.

### [@OleksandrRedko](https://github.com/alexandear-org) - Oleksandr Redko
Improved code quality by fixing typos across multiple files and filenames, simplifying format.All implementation, and replacing context.Background with t.Context in tests.

### [@lbeckman314](https://github.com/lbeckman314) - Liam Beckman
Fixed critical viewer handler bug by setting proper Content-Type headers for file downloads, enabling canvas and other file types to display correctly.

### [@omarcostahamido](https://github.com/omarcostahamido) - Omar Costa Hamido
Improved documentation by updating the export usage guide with clearer instructions and examples.

### [@yuvipanda](https://github.com/yuvipanda) - Yuvi Panda
Enhanced documentation by adding RST formatting fixes and documenting how to open developer tools for troubleshooting.

### [@kolsys](https://github.com/kolsys)
Implemented per-channel time range filtering by adding support for dump-from and dump-to parameters for individual channels.

## Individual Contributors

### [@ChrisEdwards](https://github.com/ChrisEdwards) - Chris Edwards
Fixed a critical panic issue when threads have no replies in the specified time range.

### [@errge](https://github.com/errge) - Gergely Risko
Updated Dockerfile to use golang 1.21, keeping the Docker build environment current.

### [@goretkin](https://github.com/goretkin) - Gustavo Nunes Goretkin
Documented how to run Slackdump from a repository checkout, making it easier for developers to contribute.

### [@jlmuir](https://github.com/jlmuir) - J. Lewis Muir
Fixed a typo in README.md improving documentation clarity.

### [@jhult](https://github.com/jhult) - Jonathan Hult
Fixed the emoji-failfast flag functionality.

### [@fitzyjoe](https://github.com/fitzyjoe) - Joseph Fitzgerald
Added jq example to documentation for better data processing guidance.

### [@robws](https://github.com/robws) - Rob
Improved dump.sh shebang for better shell compatibility and updated README.rst documentation.

### [@enterJazz](https://github.com/enterJazz) - Robert Schambach
Improved dump.sh shebang for better portability across different systems.

### [@rawlingsr](https://github.com/rawlingsr) - Ryan Rawlings
Fixed a typo in README.md improving documentation quality.

### [@ShlomoCode](https://github.com/ShlomoCode) - Shlomo
Updated README.rst with documentation improvements.

### [@YassineOsip](https://github.com/YassineOsip) - Yassine Lafkih
Fixed snippet examples in documentation for better clarity.

### [@eau-u4f](https://github.com/eau-u4f) - eau
Contributed the Silent logger implementation for quieter operation modes.

### [@snova-jamesv](https://github.com/snova-jamesv)
Clarified the login step in quickstart.md to reduce user confusion during initial setup.

### [@ChrisEdwards](https://github.com/ChrisEdwards) - Chris Edwards
Fix the panic in text conversion, when the thread has no replies in the time range.

---

**Total Contributors**: 25 (including dependabot)

This list is sorted by number of contributions. Every contribution, no matter how small, helps make Slackdump better for everyone!

If you'd like to contribute, please check out our [Code of Conduct](CODE_OF_CONDUCT.md) and submit a pull request.
