# Slugify

Converts a string into a slug that contains only lowercase letters, digits, and dashes.
The slug will not begin or end with a dash and will not contain runs of multiple dashes. The slug is not forced into being ASCII and may contain unicode characters, e.g. from other languages; however the slug is NFKD normalized, which breaks down characters into their compatibility equivalences e.g. ﬁ is decomposed into "f" and "i", or "ö" into "o".

Note that because the slug is not forced into ASCII, it is not technically URL safe. You can percent encode the slug to ensure it's safe; otherwise you can rely on the fact that most modern browsers handle unicode in the address bar.

## Usage

Creating slugs:

```go
slugify.Slugify("Hello, World!")
// "hello-world"

slugify.Slugifyf("%s & %s", "naïve", "pretentious")
// "naive-and-pretentious
```

Validating slugs:

```go
err := slugify.Validate("hello-world")
// nil

err = slugify.Validate("中文测试")
// nil

err = slugify.Validate("multiple----hyphens")
// slugify.ErrDashes

err = slugify.Validate("")
// slugify.ErrEmpty

err = slugify.Validate("Mixed_separators.are here!")
// slugify.ErrInvalid
```