package gravatar_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/gravatar"
)

func TestGravatar(t *testing.T) {
	email := "MyEmailAddress@example.com "
	url := gravatar.New(email, nil)
	assert.Equal(t, "https://www.gravatar.com/avatar/0bc83cb571cd1c50ba6f3e8a78ef1346?d=identicon&r=pg&s=80", url)
}

func TestGravatarOptions(t *testing.T) {
	email := "MyEmailAddress@example.com "
	url := gravatar.New(email, &gravatar.Options{Size: 128, ForceDefault: true, FileExtension: ".png"})
	assert.Equal(t, "https://www.gravatar.com/avatar/0bc83cb571cd1c50ba6f3e8a78ef1346.png?f=y&s=128", url)
}

func TestHash(t *testing.T) {
	// Test case from: https://en.gravatar.com/site/implement/hash/
	input := "MyEmailAddress@example.com "
	expected := "0bc83cb571cd1c50ba6f3e8a78ef1346"
	assert.Equal(t, expected, gravatar.Hash(input))
}
