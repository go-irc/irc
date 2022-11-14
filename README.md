# go-irc

[![GoDoc](https://img.shields.io/badge/doc-GoDoc-blue.svg)](https://pkg.go.dev/gopkg.in/irc.v4)
[![Build Status](https://img.shields.io/github/workflow/status/go-irc/irc/CI.svg)](https://github.com/go-irc/irc/actions)
[![Coverage Status](https://img.shields.io/coveralls/go-irc/irc.svg)](https://coveralls.io/github/go-irc/irc?branch=master)

This package was originally created to only handle message parsing, but has since been expanded to include small abstractions around connections and a very general client type with some small conveniences.

This library is not designed to hide any of the IRC elements from you. If you just want to build a simple chat bot and don't want to deal with IRC in particular, there are a number of other libraries which provide a more full featured client if that's what you're looking for.

This library is meant to stay as simple as possible so it can be a building block for other packages.

This library aims for API compatibility whenever possible. New functions and other additions will not result in a major version increase unless they break the API. This library aims to follow the semver recommendations mentioned on gopkg.in.

This packages uses newer error handling APIs so, only go 1.13+ is officially supported.

## Import Paths

All development happens on the `master` branch and when features are considered stable enough, a new release will be tagged.

* `gopkg.in/irc.v4` should be used to develop against the commits tagged as stable

## Development

In order to run the tests, make sure all submodules are up to date. If you are just using this library, these are not needed.

## Notes on Unstable APIs

Currently the ISupport and Tracker APIs are considered unstable - these may be broken or removed with minor version changes, so use them at your own risk.

## Major Version Changes

### v4

- Added initial ISupport and Tracker support as unstable APIs
- Drop the separate TagValue type
- Drop Tags.GetTag

### v3

- Import path changed back to `gopkg.in/irc.v3` without the version suffix.

### v2

- CTCP messages will no longer be rewritten. The decision was made that this library should pass through all messages without mangling them.
- Remove Message.FromChannel as this is not always accurate, while Client.FromChannel should always be accurate.

### v1

Initial release
