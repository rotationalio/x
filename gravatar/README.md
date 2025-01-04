# Gravatar

Creates [gravatar](https://docs.gravatar.com/) URLs from email addresses using default options.

Gravatars are a personalized user image based on the email address of the user. They
allow applications to fetch an image of the user without having to store images locally.
All gravatars are computed based on the hash of a user's email along with some options.
This package generates those URL strings for use in other applications.

Usage:

```go
url := gravatar.New("MyEmailAddress@example.com", nil)
// Output:
// https://www.gravatar.com/avatar/0bc83cb571cd1c50ba6f3e8a78ef1346?d=identicon&r=pg&s=80
```

Options are:

```go
// Options allows you to specify preferences that are added as URL query params.
type Options struct {
	// The square size of the image; an request images from 1px up to 2048px.
	Size int

	// One of 404, mp, identicon, monsterid, wavatar, retro, robohash, or blank.
	DefaultImage string

	// Force the default image to always load
	ForceDefault bool

	// Rating indicates image appropriateness, one of g, pg, r, or x.
	Rating string

	// File extension is optional, can be one of .png, .jpg, etc.
	FileExtension string
}
```