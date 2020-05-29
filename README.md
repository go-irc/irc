# go-irc

[![GoDoc](https://img.shields.io/badge/doc-GoDoc-blue.svg)](https://godoc.org/github.com/go-irc/irc)
[![Build Status](https://img.shields.io/github/workflow/status/go-irc/irc/CI.svg)](https://github.com/go-irc/irc/actions)
[![Coverage Status](https://img.shields.io/coveralls/go-irc/irc.svg)](https://coveralls.io/github/go-irc/irc?branch=master)

This package was originally created to only handle message parsing,
but has since been expanded to include a small abstraction around a
connection.

Additional abstractions can be found in the
[ircx](https://github.com/go-irc/ircx) package.

This library is not designed to hide any of the IRC elements from
you. If you just want to build a simple chat bot and don't want to
deal with IRC in particular, there are a number of other libraries
which provide a more full featured client if that's what you're
looking for.

This library is meant to stay as simple as possible so it can be a
building block for other packages.

This library aims for API compatibility whenever possible. New
functions and other additions will most likely not result in a major
version increase unless they break the API. This library aims to
follow the semver recommendations mentioned on gopkg.in.

Due to complications in how to support x/net/context vs the built-in context
package, only go 1.7+ is officially supported.

## Import Paths

All development happens on the `master` branch and when features are
considered stable enough, a new release will be tagged.

* `github.com/go-irc/irc/v4` should be used to develop against the commits
  tagged as stable

Note that this will most-likely change back to `gopkg.in/go-irc/irc.v4` once out
of pre-release status.

## Development

In order to run the tests, make sure all submodules are up to date. If you are
just using this library, these are not needed.

## Major Version Changes

### v4 - Under Development

- Move client to the new ircx package

### v3

- Import path changed back to `gopkg.in/irc.v3` without the version suffix.

### v2

- CTCP messages will no longer be rewritten. The decision was made that this
  library should pass through all messages without mangling them.
- Remove Message.FromChannel as this is not always accurate, while
  Client.FromChannel should always be accurate.

### v1

Initial release
