# Slackdump Agent Guide

This file captures project-wide conventions for agents working in this
repository.

## Go Test Naming

Keep unit tests mapped 1:1 to the function or method they test.

- For a package function `procChanMsg`, use `Test_procChanMsg`.
- For a method with a receiver, include the receiver type before the method:
  `(*Stream).thread` is tested by `TestStream_thread`.
- Put behavior variants inside that single test function as `t.Run(...)`
  subtests.
- Do not add separate top-level tests for one scenario of the same function.
  Add a new table row or subtest under the existing 1:1 test instead.

