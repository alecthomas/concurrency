# A collection of types and functions for structured concurrency

[![PkgGoDev](https://pkg.go.dev/badge/github.com/alecthomas/concurrency)](https://pkg.go.dev/github.com/alecthomas/concurrency) [![GHA Build](https://github.com/alecthomas/concurrency/actions/workflows/ci.yml/badge.svg)](https://github.com/alecthomas/concurrency/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/alecthomas/concurrency)](https://goreportcard.com/report/github.com/alecthomas/concurrency) [![Slack chat](https://img.shields.io/static/v1?logo=slack&style=flat&label=slack&color=green&message=gophers)](https://gophers.slack.com/messages/CN9DS8YF3)

This package centres around a managed tree of goroutines. An error in any
goroutine or sub-tree will cancel the entire tree. Each node in the tree is a
group of goroutines that can be waited on.

There are also useful concurrent functions that utilise the tree.
