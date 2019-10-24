# Toolkit [![GoDoc](https://godoc.org/github.com/kansaslabs/x?status.svg)](https://godoc.org/github.com/kansaslabs/x)

**Go packages that are common to many kansaslabs projects -- in the spirit of golang.org/x**

## Usage

To get these packages into your project, it's as easy as:

    $ go get github.com/kansaslabs/x/[pkg]

Where `[pkg]` is the name of hte package you want to use in your project. Note that the go modules are at the top level of the toolkit, so please specify the latest version of the `x` package that has the tools that you need.

## Subpackages

This is single repository that stores many, independent small subpackages. This list changes often as common code gets moved from specific projects into this repository.

- [out](out/): hierarchical logger to manage logging verbosity to stdout
- [noplog](noplog/): no operation logger to capture internal logging with no output

## About

Package x hosts several packages, modules, and libraries that are common across most kansaslabs projects for easy reuse. This package is very much in the spirit of [golang.org/x](https://godoc.org/-/subrepo) though it does have a slightly longer import path as a result of being hosted in a GitHub repository.

It is important to note is that most of the subpackages in this repository are independent. That is that they are implemented and tested separately from other subpackages. Anyone who would like to use this package should only go get exactly what they need and rely on the documentation on godoc and in the subpackage README.md for more information.
