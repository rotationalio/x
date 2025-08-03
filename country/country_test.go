package country_test

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/country"
)

const testdata = "testdata/countries.json.gz"

func TestRealCountries(t *testing.T) {
	countries, err := readCountriesData()
	assert.Ok(t, err)

	t.Run("List Matches", func(t *testing.T) {
		all := country.Countries()
		assert.Equal(t, len(countries), len(all), "should have the same number of countries")
	})

	t.Run("Lookups", func(t *testing.T) {
		for _, c := range countries {
			f, err := country.Lookup(c.Alpha2)
			assert.Ok(t, err, "should be able to lookup country by Alpha2 code")
			assert.Equal(t, c, f, "should find correct country by Alpha2 code")

			f, err = country.Lookup(c.Alpha3)
			assert.Ok(t, err, "should be able to lookup country by Alpha3 code")
			assert.Equal(t, c, f, "should find correct country by Alpha3 code")

			f, err = country.Lookup(c.ShortName)
			assert.Ok(t, err, "should be able to lookup country by ShortName")
			assert.Equal(t, c, f, "should find correct country by ShortName")

			f, err = country.Lookup(c.LongName)
			assert.Ok(t, err, "should be able to lookup country by LongName")
			assert.Equal(t, c, f, "should find correct country by LongName")

			for _, name := range c.UnofficialNames {
				f, ok := country.Find(name)
				assert.True(t, ok, "should find country by UnofficialName")
				assert.Equal(t, c, f, "should find correct country by UnofficialName")
			}
		}
	})

	t.Run("Flag", func(t *testing.T) {
		for _, c := range countries {
			assert.NotEqual(t, "", c.Flag(), "should have a non-empty flag for country %s", c.Alpha2)
		}
	})
}

func TestLookupNotFound(t *testing.T) {
	cases := []string{
		"b", "B", "xx", "XX", "zzz", "ZZZ", "Unknown Country", "NonExistent",
	}

	for _, c := range cases {
		_, err := country.Lookup(c)
		assert.ErrorIs(t, err, country.ErrNotFound, "should not find country")
	}
}

func TestLookupInvalidCode(t *testing.T) {
	t.Run("Alpha2", func(t *testing.T) {
		testCases := []string{
			"", "  ", " A", "B ", "1", "A", "AAA", "aB", "aB1", "A1B", "A B",
			"AB ", "AB1", "AB2C", "ABCD", "AB12", "A B C",
		}

		for _, code := range testCases {
			_, err := country.Alpha2(code)
			assert.ErrorIs(t, err, country.ErrInvalidCode, "should return invalid code error for Alpha2")
		}
	})

	t.Run("Alpha3", func(t *testing.T) {
		testCases := []string{
			"", "   ", "1", "A", "AA", "AB ", " AB", "aBc", "aB1", "A1B", "A B",
			"AB ", "AB1", "AB2C", "ABCD", "AB12", "A B C",
		}

		for _, code := range testCases {
			_, err := country.Alpha3(code)
			assert.ErrorIs(t, err, country.ErrInvalidCode, "should return invalid code error for Alpha3")
		}
	})
}

func TestFlag(t *testing.T) {
	// Spot check a few flags
	cases := []struct {
		alpha2   string
		expected string
	}{
		{"US", "🇺🇸"},
		{"GB", "🇬🇧"},
		{"FR", "🇫🇷"},
		{"DE", "🇩🇪"},
		{"JP", "🇯🇵"},
	}

	for _, c := range cases {
		flag, err := country.Flag(c.alpha2)
		assert.Ok(t, err, "should not error for valid Alpha2 code %s", c.alpha2)
		assert.Equal(t, c.expected, flag, "should return correct flag for %s", c.alpha2)
	}
}

// Read Countries JSON data from testdata/countries.json.gz
func readCountriesData() (out []*country.Country, err error) {
	var f *os.File
	if f, err = os.Open(testdata); err != nil {
		return nil, fmt.Errorf("could not open %s: %w", testdata, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("could not create gzip reader: %w", err)
	}
	defer gz.Close()

	if err = json.NewDecoder(gz).Decode(&out); err != nil {
		return nil, fmt.Errorf("could not decode JSON data: %w", err)
	}

	return out, nil
}
