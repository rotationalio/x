package slugify_test

import (
	"errors"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/slugify"
)

var testCases = []struct{ input, expected string }{
	{"", ""},
	{"     ", ""},
	{"!!!", ""},
	{"Hello, World!", "hello-world"},
	{"Go is awesome", "go-is-awesome"},
	{"I'm a good developer", "i-m-a-good-developer"},
	{"This is a test.", "this-is-a-test"},
	{">simple test<---", "simple-test"},
	{"underscores_are_cool", "underscores-are-cool"},
	{"Dots.are.here", "dots-are-here"},
	{"Mixed_separators.are here!", "mixed-separators-are-here"},
	{"Café au lait", "cafe-au-lait"},
	{"100% sure!", "100-percent-sure"},
	{"Go@2024", "go-at-2024"},
	{"Multiple   spaces", "multiple-spaces"},
	{"Special !#$^*(){}[]></?'\",.:;+=_`~ characters", "special-characters"},
	{"Trailing spaces   ", "trailing-spaces"},
	{"   Leading spaces", "leading-spaces"},
	{"Mixed CASE Input", "mixed-case-input"},
	{"Emoji 😊 test", "emoji-test"},
	{"中文测试", "中文测试"},
	{"naïve approach", "naive-approach"},
	{"coöperate", "cooperate"},
	{"rock & roll", "rock-and-roll"},
	{"123456", "123456"},
	{"version 2.0.1", "version-2-0-1"},
	{"日本語の手紙をテスト", "日本語の手紙をテスト"},
	{"-no leading hyphen", "no-leading-hyphen"},
	{"no trailing hyphen-", "no-trailing-hyphen"},
	{"----leading-hyphens", "leading-hyphens"},
	{"multiple----hyphens", "multiple-hyphens"},
	{"trailing-hyphens----", "trailing-hyphens"},
}

func TestSlugify(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := slugify.Slugify(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestSlugifyf(t *testing.T) {
	slug := slugify.Slugifyf("Hello, %s - %d times!", "World", 3)
	assert.Equal(t, "hello-world-3-times", slug)
}

func TestValidate(t *testing.T) {
	// Skip the first three test cases which return empty strings.
	for _, tc := range testCases[3:] {
		t.Run(tc.expected, func(t *testing.T) {
			err := slugify.Validate(tc.expected)
			assert.Ok(t, err)
		})
	}

	t.Run("Empty", func(t *testing.T) {
		assert.ErrorIs(t, slugify.Validate(""), slugify.ErrEmpty)
	})

	t.Run("Invalid", func(t *testing.T) {
		for i, tc := range testCases[3:] {
			if tc.input == tc.expected {
				continue
			}

			err := slugify.Validate(tc.input)
			assert.True(t, errors.Is(err, slugify.ErrInvalid) || errors.Is(err, slugify.ErrDashes), "expected error on test case %d", i)
		}
	})
}
