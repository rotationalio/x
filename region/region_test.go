package region_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/region"
)

type parseTestCase struct {
	input    any
	expected region.Region
}

func TestParse(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []parseTestCase{}

		// Create test cases from the data file
		for name, info := range iterateData(t) {
			tests = append(tests, parseTestCase{
				input:    region.Denormalize(name),
				expected: info.ID,
			}, parseTestCase{
				input:    region.Normalize(name),
				expected: info.ID,
			}, parseTestCase{
				input:    fmt.Sprintf("%d", info.ID),
				expected: info.ID,
			}, parseTestCase{
				input:    info.ID.String(),
				expected: info.ID,
			}, parseTestCase{
				input:    int(info.ID),
				expected: info.ID,
			}, parseTestCase{
				input:    int32(info.ID),
				expected: info.ID,
			}, parseTestCase{
				input:    int64(info.ID),
				expected: info.ID,
			}, parseTestCase{
				input:    uint(info.ID),
				expected: info.ID,
			}, parseTestCase{
				input:    uint32(info.ID),
				expected: info.ID,
			}, parseTestCase{
				input:    uint64(info.ID),
				expected: info.ID,
			}, parseTestCase{
				input:    float32(info.ID),
				expected: info.ID,
			}, parseTestCase{
				input:    float64(info.ID),
				expected: info.ID,
			}, parseTestCase{
				input:    json.Number(fmt.Sprintf("%d", info.ID)),
				expected: info.ID,
			})
		}

		for i, test := range tests {
			t.Run(fmt.Sprintf("Case %d: %+v", i, test.input), func(t *testing.T) {
				actual, err := region.Parse(test.input)
				assert.Ok(t, err)
				assert.Equal(t, test.expected, actual)
			})
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []parseTestCase{
			{input: ""},
			{input: nil},
			{input: true},
			{input: 13239},
			{input: -13239},
			{input: json.Number("foo")},
		}

		for i, test := range tests {
			t.Run(fmt.Sprintf("Case %d: %+v", i, test.input), func(t *testing.T) {
				actual, err := region.Parse(test.input)
				assert.NotNil(t, err, "expected an error to be returned")
				assert.Equal(t, region.UNKNOWN, actual)
			})
		}
	})
}

func TestList(t *testing.T) {
	count := -1 // -1 to exclude the UNKNOWN region
	for range iterateData(t) {
		count++
	}

	assert.True(t, count > 0, "expected at least one region to be listed")

	regions := region.List()
	assert.Equal(t, count, len(regions), "expected the number of regions to be the same")
}

func TestString(t *testing.T) {
	for _, r := range region.List() {
		s := r.String()
		p, err := region.Parse(s)
		assert.Ok(t, err)
		assert.Equal(t, r, p)
	}
}

func TestStringUnknown(t *testing.T) {
	r := region.Region(934134345)
	assert.NotEqual(t, region.UNKNOWN, r)
	assert.Equal(t, "unknown", r.String())
}

func TestJSON(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		for _, r := range region.List() {
			s, err := json.Marshal(r)
			assert.Ok(t, err)

			var p region.Region
			assert.Ok(t, json.Unmarshal(s, &p))
			assert.Equal(t, r, p)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := make([][]byte, 0, 3)
		tests = append(tests, []byte{})
		tests = append(tests, []byte{0x00})
		tests = append(tests, []byte(`"foo`))
		tests = append(tests, []byte(`true`))
		tests = append(tests, []byte(`1234567890`))

		for _, tc := range tests {
			var p region.Region
			assert.NotNil(t, p.UnmarshalJSON(tc), "expected an error to be returned")
		}
	})
}

func TestBinary(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		for _, r := range region.List() {
			s, err := r.MarshalBinary()
			assert.Ok(t, err)

			var p region.Region
			assert.Ok(t, p.UnmarshalBinary(s))
			assert.Equal(t, r, p)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := make([][]byte, 0, 3)
		tests = append(tests, []byte(`"foo`))
		tests = append(tests, []byte(`true`))
		tests = append(tests, []byte(`1234567890`))

		for _, tc := range tests {
			var p region.Region
			assert.NotNil(t, p.UnmarshalBinary(tc), "expected an error to be returned")
		}
	})
}

func TestCountries(t *testing.T) {
	counts := make(map[string]int)
	for _, r := range region.List() {
		switch r {
		case region.UNKNOWN, region.LOCALHOST, region.DEVELOPMENT, region.TESTING, region.STAGING:
			continue
		}

		info := r.Info()
		assert.NotNil(t, info, "expected a region info to be returned for %s", r)
		counts[info.CountryCode]++

		assert.Ok(t, info.Validate())

		country := info.Country()
		assert.NotNil(t, country, "expected a country to be returned for %s", r)
	}

	for country, count := range counts {
		t.Logf("%s: %d", country, count)
	}

	assert.True(t, len(counts) > 0, "expected at least one country to be found")
}

type iterator func(func(name string, region *region.Info) bool)

func iterateData(t *testing.T) iterator {
	return func(yield func(name string, region *region.Info) bool) {
		entries, err := os.ReadDir("testdata")
		assert.Ok(t, err)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			data, err := readData(filepath.Join("testdata", entry.Name()))
			assert.Ok(t, err)

			for name, region := range data {
				if !yield(name, region) {
					return
				}
			}
		}
	}
}

func readData(path string) (data map[string]*region.Info, err error) {
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return nil, fmt.Errorf("could not open %s: %w", path, err)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("could not decode JSON data: %w", err)
	}
	return data, nil
}
