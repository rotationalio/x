package typecase_test

import (
	"strings"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/typecase"
)

func TestTypecase(t *testing.T) {
	tests := []string{
		"",
		"CamelCaseAlready",
		"lowerCamelCaseAlready",
		"snake_case_already",
		"kebab-case-already",
		"CONSTANT_CASE_ALREADY",
		"Title Case Already",
		"all lower already",
		"ALL UPPER ALREADY",
		"dot.separated.already",
		"  too    many    spaces    in between  ",
		"UserID",
		"OriginatingVASP",
		"HTTPS",
		"aMix_of_CASE-formatStrings",
		"remove_punctuation!@#%&*()?",
		"fixture",
		"d e f",
		"7 8 9",
		"uJSON456",
		"global_ulid",
		"mix888digits",
		"weird---__.__---split",
		"Emoji🚀🧜🐡Fun",
		"ÜppigkeitKönigGlückFuß",
		"JSONMessage",
	}

	t.Run("Camel", func(t *testing.T) {
		expected := []string{
			"",
			"CamelCaseAlready",
			"LowerCamelCaseAlready",
			"SnakeCaseAlready",
			"KebabCaseAlready",
			"ConstantCaseAlready",
			"TitleCaseAlready",
			"AllLowerAlready",
			"AllUpperAlready",
			"DotSeparatedAlready",
			"TooManySpacesInBetween",
			"UserID",
			"OriginatingVASP",
			"HTTPS",
			"AMixOfCaseFormatStrings",
			"RemovePunctuation",
			"Fixture",
			"DEF",
			"789",
			"UJSON456",
			"GlobalULID",
			"Mix888Digits",
			"WeirdSplit",
			"Emoji🚀🧜🐡Fun",
			"ÜppigkeitKönigGlückFuß",
			"JSONMessage",
		}

		for i, test := range tests {
			actual := typecase.Camel(test)
			assert.Equal(t, expected[i], actual, "test case %d failed", i)
		}
	})

	t.Run("LowerCamel", func(t *testing.T) {
		expected := []string{
			"",
			"camelCaseAlready",
			"lowerCamelCaseAlready",
			"snakeCaseAlready",
			"kebabCaseAlready",
			"constantCaseAlready",
			"titleCaseAlready",
			"allLowerAlready",
			"allUpperAlready",
			"dotSeparatedAlready",
			"tooManySpacesInBetween",
			"userID",
			"originatingVASP",
			"https",
			"aMixOfCaseFormatStrings",
			"removePunctuation",
			"fixture",
			"dEF",
			"789",
			"uJSON456",
			"globalULID",
			"mix888Digits",
			"weirdSplit",
			"emoji🚀🧜🐡Fun",
			"üppigkeitKönigGlückFuß",
			"jsonMessage",
		}

		for i, test := range tests {
			actual := typecase.LowerCamel(test)
			assert.Equal(t, expected[i], actual, "test case %d failed", i)
		}
	})

	t.Run("Snake", func(t *testing.T) {
		expected := []string{
			"",
			"camel_case_already",
			"lower_camel_case_already",
			"snake_case_already",
			"kebab_case_already",
			"constant_case_already",
			"title_case_already",
			"all_lower_already",
			"all_upper_already",
			"dot_separated_already",
			"too_many_spaces_in_between",
			"user_id",
			"originating_vasp",
			"https",
			"a_mix_of_case_format_strings",
			"remove_punctuation",
			"fixture",
			"d_e_f",
			"7_8_9",
			"u_json_456",
			"global_ulid",
			"mix_888_digits",
			"weird_split",
			"emoji_🚀🧜🐡_fun",
			"üppigkeit_könig_glück_fuß",
			"json_message",
		}

		for i, test := range tests {
			actual := typecase.Snake(test)
			assert.Equal(t, expected[i], actual, "test case %d failed", i)
		}
	})

	t.Run("Kebab", func(t *testing.T) {
		expected := []string{
			"",
			"camel-case-already",
			"lower-camel-case-already",
			"snake-case-already",
			"kebab-case-already",
			"constant-case-already",
			"title-case-already",
			"all-lower-already",
			"all-upper-already",
			"dot-separated-already",
			"too-many-spaces-in-between",
			"user-id",
			"originating-vasp",
			"https",
			"a-mix-of-case-format-strings",
			"remove-punctuation",
			"fixture",
			"d-e-f",
			"7-8-9",
			"u-json-456",
			"global-ulid",
			"mix-888-digits",
			"weird-split",
			"emoji-🚀🧜🐡-fun",
			"üppigkeit-könig-glück-fuß",
			"json-message",
		}

		for i, test := range tests {
			actual := typecase.Kebab(test)
			assert.Equal(t, expected[i], actual, "test case %d failed", i)
		}
	})

	t.Run("Constant", func(t *testing.T) {
		expected := []string{
			"",
			"CAMEL_CASE_ALREADY",
			"LOWER_CAMEL_CASE_ALREADY",
			"SNAKE_CASE_ALREADY",
			"KEBAB_CASE_ALREADY",
			"CONSTANT_CASE_ALREADY",
			"TITLE_CASE_ALREADY",
			"ALL_LOWER_ALREADY",
			"ALL_UPPER_ALREADY",
			"DOT_SEPARATED_ALREADY",
			"TOO_MANY_SPACES_IN_BETWEEN",
			"USER_ID",
			"ORIGINATING_VASP",
			"HTTPS",
			"A_MIX_OF_CASE_FORMAT_STRINGS",
			"REMOVE_PUNCTUATION",
			"FIXTURE",
			"D_E_F",
			"7_8_9",
			"U_JSON_456",
			"GLOBAL_ULID",
			"MIX_888_DIGITS",
			"WEIRD_SPLIT",
			"EMOJI_🚀🧜🐡_FUN",
			"ÜPPIGKEIT_KÖNIG_GLÜCK_FUß",
			"JSON_MESSAGE",
		}

		for i, test := range tests {
			actual := typecase.Constant(test)
			assert.Equal(t, expected[i], actual, "test case %d failed", i)
		}
	})

	t.Run("Title", func(t *testing.T) {
		expected := []string{
			"",
			"Camel Case Already",
			"Lower Camel Case Already",
			"Snake Case Already",
			"Kebab Case Already",
			"Constant Case Already",
			"Title Case Already",
			"All Lower Already",
			"All Upper Already",
			"Dot Separated Already",
			"Too Many Spaces In Between",
			"User ID",
			"Originating VASP",
			"HTTPS",
			"A Mix Of Case Format Strings",
			"Remove Punctuation",
			"Fixture",
			"D E F",
			"7 8 9",
			"U JSON 456",
			"Global ULID",
			"Mix 888 Digits",
			"Weird Split",
			"Emoji 🚀🧜🐡 Fun",
			"Üppigkeit König Glück Fuß",
			"JSON Message",
		}

		for i, test := range tests {
			actual := typecase.Title(test)
			assert.Equal(t, expected[i], actual, "test case %d failed", i)
		}
	})
}

func TestJoin(t *testing.T) {
	tests := []struct {
		parts      []string
		sep        string
		formatters []typecase.Formatter
		expected   string
	}{
		{
			parts:      []string{"hello", "world"},
			sep:        "_",
			formatters: []typecase.Formatter{strings.ToUpper},
			expected:   "HELLO_WORLD",
		},
		{
			parts:      []string{"hello", "", "world", ""},
			sep:        "_",
			formatters: []typecase.Formatter{strings.ToUpper},
			expected:   "HELLO_WORLD",
		},
		{
			parts:      []string{"hello", "", "world", ""},
			sep:        "-",
			formatters: []typecase.Formatter{},
			expected:   "hello-world",
		},
	}

	for i, tc := range tests {
		actual := typecase.Join(tc.parts, tc.sep, tc.formatters...)
		assert.Equal(t, tc.expected, actual, "test case %d failed", i)
	}
}
