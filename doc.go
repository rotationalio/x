/*
Package x hosts several packages, modules, and libraries that are common across most
kansaslabs projects for easy reuse. This package is very much in the spirit of
golang.org/x though it does have a slightly longer import path as a result of being
hosted in a GitHub repository.

It is important to note is that most of the subpackages in this repository are
independent. That is that they are implemented and tested separately from other
subpackages. Anyone who would like to use this package should only go get exactly what
they need and rely on the documentation on godoc and in the subpackage README.md for
more information.
*/
package x
