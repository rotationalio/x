# Typecase

Allows you to switch between CamelCase, lowerCamelCase, snake_case, kebab-case, and CONSTANT_CASE. This code is generally used in our template functions for HTML, code generation tools, or for portability between langauges or different JSON representations of keys.

Features:

- Handles unicode
- Keeps common acronyms and parts together such as `ID` or `HTTP`

## Usage

```go
import (
    "fmt"

    "go.rtnl.x/typecase"
)

input := "a_StringTo TYPE_CASE"

// Convert to different cases!
fmt.Println(casing.Camel(input))
fmt.Println(casing.LowerCamel(input))
fmt.Println(casing.Snake(input))
```

## Inspiration

Thanks to the following libraries for inspiration:

- [danielgtaylor/casing](https://github.com/danielgtaylor/casing)
- [gobeam/stringy](https://github.com/gobeam/stringy)
