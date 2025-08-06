# Countries

Resolves [ISO 3166-1](https://en.wikipedia.org/wiki/ISO_3166-1) country alpha codes to country information such as name, flag, etc. This package is primarily used on the front end to display country names, flag emojis, or languages; and on the backend to identify country data with a simple code enumeration.

## Usage

To lookup country data by an ISO Alpha-2 or Alpha-3 code you can:

```go
france, err := country.Alpha2("FR")
peru, err := country.Alpha2("PER")
```

To search for a country by name:

```go
vietnam, ok := country.Find("ベトナム")
```

NOTE: when you find by country name for the first time a Trie with all of the country names is built into memory, enabling fast lookups.

To search by alpha code or by name, you can use lookup:

```go
country, err := country.Lookup("BV")
```

To get an emoji flag for a country by the Alpha-2 code:

```go
country.Flag("JP")
// 🇯🇵
```

Country data is as follows:

```go
type Country struct {
	Alpha2          string   `json:"alpha2"`
	Alpha3          string   `json:"alpha3"`
	ShortName       string   `json:"iso_short_name"`
	LongName        string   `json:"iso_long_name"`
	CurrencyCode    string   `json:"currency_code"`
	DistanceUnit    string   `json:"distance_unit"`
	UnofficialNames []string `json:"unofficial_names"`
	Region          string   `json:"world_region"`
	Subregion       string   `json:"subregion"`
	Continent       string   `json:"continent"`
	Languages       []string `json:"languages_spoken"`
}
```

Additional fields are available and can be populated with a new PR. See Generation below.

## Generation

The data populating the country codes comes from [countries-data-json](https://github.com/countries/countries-data-json) which is automatically updated from the `countries` Ruby gem. To download the latest version of the country files and recompile the `countries.data.go` file, simply run `go generate ./...` at the top level of the `x` repository.

## Benchmarks

One of the primary considerations of this package is how to perform lookups of country codes or names to the full country data structures. An obvious solution is a map; however maps induce some overhead with hashes and memory allocation. For the ISO-3166-1 Alpha 2 codes, we have a fixed space of 676 possible options. As a reuslt, we use a lookup table which improves upon the use of the map fairly significantly and has less memory overhead.

```
goos: darwin
goarch: arm64
pkg: go.rtnl.ai/x/country
cpu: Apple M1 Max
BenchmarkTable2Lookup-10    	167973045	         6.967 ns/op	       0 B/op	       0 allocs/op
BenchmarkMap2Lookup-10      	65298570	        18.25 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrie2Lookup-10     	100000000	        10.63 ns/op	       0 B/op	       0 allocs/op
```

The ISO-3166-1 Alpha 3 lookups are slighly larger as there are 17,576 options. Because of this, we explored using a Trie data structure; which did improve upon the performance of the map lookup. However, it did not significantly decrease the amount of space used by the lookup table, therefore we are also using a lookup table for Alpha 3 values.

```
goos: darwin
goarch: arm64
pkg: go.rtnl.ai/x/country
cpu: Apple M1 Max
BenchmarkTable3Lookup-10    	238214187	         5.077 ns/op	       0 B/op	       0 allocs/op
BenchmarkMap3Lookup-10      	44839419	        25.86 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrie3Lookup-10     	158254705	         7.564 ns/op	       0 B/op	       0 allocs/op
```

See commit [0bb88fa](https://github.com/rotationalio/x/commit/0bb88fadf000eb1b7aabf48c9b2ad3ac8dc0ce2b) for the code used to generate and execute the above benchmarks.