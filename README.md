# Toolkit [![GoDoc](https://godoc.org/go.rtnl.ai/x?status.svg)](https://godoc.org/go.rtnl.ai/x)

**Go packages that are common to many Rotational Labs projects -- in the spirit of golang.org/x**

## Usage

To get these packages into your project, it's as easy as:

    $ go get go.rtnl.ai/x/[pkg]

Where `[pkg]` is the name of the package you want to use in your project. Note that the go modules are at the top level of the toolkit, so please specify the latest version of the `x` package that has the tools that you need.

## Subpackages

This is single repository that stores many, independent small subpackages. This list changes often as common code gets moved from specific projects into this repository.

- [assert](https://go.rtnl.ai/x/assert): simple test assertions for no-dependency testing
- [out](https://go.rtnl.ai/x/out): hierarchical logger to manage logging verbosity to stdout
- [noplog](https://go.rtnl.ai/x/noplog): no operation logger to capture internal logging with no output
- [probez](https://go.rtnl.ai/x/probez): http handlers for kubernetes probes (livez, healthz, and readyz)
- [gravatar](https://go.rtnl.ai/x/gravatar): helper to create Gravatar urls from email addresses
- [humanize](https://go.rtnl.ai/x/humanize): creates human readable strings from various types
- [base58](https://go.rtnl.ai/x/base58): base58 encoding package as used by Bitcoin and travel addresses
- [randstr](https://go.rtnl.ai/x/randstr): generate random strings using the crypto/rand package as efficiently as possible
- [api](https://go.rtnl.ai/x/api): common utilities and responses for our JSON/REST APIs that our services run.
- [dsn](https://go.rtnl.ai/x/dsn): parses data source names in order to connect to both server and embedded databases easily.
- [semver](https://go.rtnl.ai/x/semver): allows parsing and comparison of semantic versioning numbers.

## About

Package x hosts several packages, modules, and libraries that are common across most Rotational Labs projects for easy reuse. This package is very much in the spirit of [golang.org/x](https://godoc.org/-/subrepo) and even has a vanity url to make the import path as short as possible!

It is important to note is that most of the subpackages in this repository are independent. That is that they are implemented and tested separately from other subpackages. Anyone who would like to use this package should only go get exactly what they need and rely on the documentation on godoc and in the subpackage README.md for more information.
