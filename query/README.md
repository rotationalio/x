# Query

Encodes structs into URL query parameters. This is a port of [github.com/google/go-querystring](https://github.com/google/go-querystring).

## Usage

```go
import "go.rtnl.ai/x/query"
```

This package allows you to construct a URL using a struct that represents URL query parameters and to enforce the type safety of those parameters. This is done through a `query.Values()` function that reads struct tags to create the raw query as shown below:

```go
type PageQuery struct {
    Page int `url:"page"`
    Size int `url:"page_size"`
    Order string `url:"order"`
    Archives bool `url:"archives"`
}

query := PageQuery{ 4, 100, "asc", true }
v, _ := query.Values(opt)
fmt.Print(v.Encode())
// page=4&page_size=100&order=asc&archives=true
```

Currently this package only performs encoding of query values, not decoding. Generally speaking we use `gin.BindQuery` for decoding values in our web applications.
