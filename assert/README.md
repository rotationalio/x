# Assertion Helpers

Because this is a library, we prefer to have no dependencies including our usual test
dependencies (e.g. testify require). So we have some basic assertion helpers for tests.

See: https://github.com/benbjohnson/testing

Usage:

```go
import "go.rtnl.ai/x/assert"

func TestMyFunc(t *testing.T) {
    // Basic assertion of a condition
    assert.Assert(t, 2+2 == 4, "ensure addition works")

    // Assert true or false
    assert.True(t, 2+2 == 4, "should be equal")
    assert.False(t, 2+1 == 4, "should not be equal")

    // For a function that returns an error, ensure no error is returned
    assert.Ok(t, MyFunc(), "expected error to be nil")

    // Test equality
    assert.Equals(t, expected, actual)

    // Test the error is a specific error target
    assert.ErrorIs(t, err, MyError)
}
```