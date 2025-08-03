package country_test

import (
	"math/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/country"
	"go.rtnl.ai/x/randstr"
)

func TestTrie(t *testing.T) {
	countries, err := readCountriesData()
	assert.Ok(t, err)
	assert.Len(t, countries, 249, "the countries test data has changed, please update this test")

	// Should be able to insert all countries into the trie
	root := &Trie{}
	for _, country := range countries {
		root.Insert(country.Alpha2, country)
		root.Insert(country.Alpha3, country)
		root.Insert(country.ShortName, country)
		root.Insert(country.LongName, country)
		for _, name := range country.UnofficialNames {
			root.Insert(name, country)
		}
	}

	// Should be able to find all countries in the trie
	for _, country := range countries {
		found, ok := root.Find(country.Alpha2)
		assert.True(t, ok, "should find country by Alpha2 code")
		assert.Equal(t, country, found, "should find correct country by Alpha2 code")

		found, ok = root.Find(country.Alpha3)
		assert.True(t, ok, "should find country by Alpha3 code")
		assert.Equal(t, country, found, "should find correct country by Alpha3 code")

		found, ok = root.Find(country.ShortName)
		assert.True(t, ok, "should find country by ShortName")
		assert.Equal(t, country, found, "should find correct country by ShortName")

		found, ok = root.Find(country.LongName)
		assert.True(t, ok, "should find country by LongName")
		assert.Equal(t, country, found, "should find correct country by LongName")

		for _, name := range country.UnofficialNames {
			found, ok = root.Find(name)
			assert.True(t, ok, "should find country by UnofficialName")
			assert.Equal(t, country, found, "should find correct country by UnofficialName")
		}
	}

	// Should not be able to find random strings in the trie
	for i := 0; i < 32; i++ {
		name := randstr.Alpha(rand.Intn(10) + 3)
		found, ok := root.Find(name)
		assert.False(t, ok, "should not find country by random name")
		assert.Nil(t, found, "should not find country by random name")
	}
}
