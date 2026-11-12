# Humanize

Utilities to convert various types into more human readable strings.

```go
import "go.rtnl.ai/x/humanize"
```

The generic functions accept any type that satisfies the package constraints (`Numeric`, `Integer`, `Signed`, `Unsigned`, `Float`, `String`), including named types such as `type Celsius float64`.

## Numbers

`Comma` adds a comma after every three orders of magnitude to any integer or float. `Commaf` does the same for floats with a fixed number of decimal places, truncating (not rounding) or padding with zeros as necessary:

```go
humanize.Comma(1234567)       // "1,234,567"
humanize.Comma(-1234.5678)    // "-1,234.5678"
humanize.Commaf(1234.5678, 2) // "1,234.56"
humanize.Commaf(1234.5, 3)    // "1,234.500"
humanize.Commaf(1234.5678, 0) // "1,234"
```

`ParseComma` and `ParseCommaf` convert those strings back into an `int64` or a `float64`:

```go
n, err := humanize.ParseComma("1,234,567") // 1234567
f, err := humanize.ParseCommaf("1,234.56") // 1234.56
```

`Ftoa` formats a float without commas, truncating it to the specified number of decimal places (at most 6) and removing any trailing zeros:

```go
humanize.Ftoa(3.14159, 2) // "3.14"
humanize.Ftoa(1.5, 3)     // "1.5"
humanize.Ftoa(2.0, 2)     // "2"
```

## Bytes

`FromBytes` returns the largest binary unit (`B`, `KiB`, `MiB`, `GiB`, or `TiB`) for a number of bytes along with the value in that unit:

```go
units, value := humanize.FromBytes(1572864) // "MiB", 1.5
fmt.Println(humanize.Ftoa(value, 2), units) // 1.5 MiB
```

The units are also available as the `ByteUnit` constants `Bytes`, `Kilobytes`, `Megabytes`, `Gigabytes`, and `Terabytes`; each is the number of bytes in the unit and its `String` method returns the abbreviation.

## Plurals

`Singular` reports if a quantity is exactly 1 and `Plural` uses it to choose between the singular and plural forms that you supply:

```go
humanize.Singular(1)                 // true
humanize.Plural(1, "goose", "geese") // "goose"
humanize.Plural(3, "goose", "geese") // "geese"
```

`MakePlural` builds the plural form from the singular unit, a suffix, and a replacement when the quantity is not 1:

```go
humanize.MakePlural(2, "byte", "", "")       // "bytes": appends "s" by default
humanize.MakePlural(2, "bus", "es", "")      // "buses": appends the suffix
humanize.MakePlural(2, "goose", "", "geese") // "geese": uses the replacement
humanize.MakePlural(2, "wolf", "ves", "f")   // "wolves": replaces the "f" ending with the suffix
```

`Pluralize` makes a best effort guess at the plural using English spelling rules; it cannot handle irregular plurals:

```go
humanize.Pluralize("box")  // "boxes"
humanize.Pluralize("city") // "cities"
```

## Lists

`AndList` and `OrList` join a slice of any string type into a natural language list; `NaturalList` joins a `[]string` with any conjunction:

```go
humanize.AndList([]string{"red", "green", "blue"})            // "red, green and blue"
humanize.OrList([]string{"red", "green"})                     // "red or green"
humanize.NaturalList([]string{"red", "green", "blue"}, "nor") // "red, green nor blue"
```

## Times

`Time` describes a timestamp relative to the current time:

```go
humanize.Time(time.Now().Add(-5 * time.Minute)) // "5 minutes ago"
humanize.Time(time.Now().Add(90 * time.Minute)) // "1 hour from now"
```

`Since` and `Until` describe the time between two timestamps using your own label. The first timestamp must be before the second for `Since` and after the second for `Until`:

```go
humanize.Since(start, now, "ago")         // "3 days ago" when start is before now
humanize.Until(deadline, now, "from now") // "1 hour from now" when deadline is after now
```

`RelativeTime` is the general form: it takes a label for each direction and the `TimePeriod` thresholds, sorted by `D`, that determine the format. Each period is used for durations shorter than `D`; in its `Format`, `%d` is the duration divided by `Den` and `%s` is the label:

```go
periods := []humanize.TimePeriod{
    {D: time.Minute, Format: "just now", Den: 1},
    {D: time.Hour, Format: "%dm %s", Den: time.Minute},
    {D: humanize.Day, Format: "%dh %s", Den: time.Hour},
}

humanize.RelativeTime(start, now, "ago", "left", periods)    // "72h ago"
humanize.RelativeTime(deadline, now, "ago", "left", periods) // "1h left"
```

The `Day`, `Week`, `Month` (30 days), `Year` (12 months), and `LongTime` (37 years) durations are available for defining periods.
