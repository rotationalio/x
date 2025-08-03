package country

import (
	"errors"
	"strings"
)

// Country data stored in-memory for fast lookups and usage inside of applications.
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
	flag            string   `json:"-"`
}

var (
	ErrNotFound    = errors.New("country not found")
	ErrInvalidCode = errors.New("invalid country code")
)

// Returns a list of all countries in the database.
func Countries() []*Country {
	var countries []*Country
	for _, country := range alpha2Lookup {
		for _, c := range country {
			if c != nil {
				countries = append(countries, c)
			}
		}
	}
	return countries
}

// Lookups the country by its Alpha-2 or Alpha-3 code depending on the input length,
// or searches for the country by its name (including unofficial names). If the country
// is not found, it returns a not found error.
func Lookup(country string) (*Country, error) {
	switch len(country) {
	case 2:
		country = strings.ToUpper(country)
		return Alpha2(country)
	case 3:
		country = strings.ToUpper(country)
		return Alpha3(country)
	default:
		country, ok := Find(country)
		if !ok {
			return nil, ErrNotFound
		}
		return country, nil
	}
}

// Fast lookup for a country by its Alpha-2 code. If the code is not two uppercase
// characters, it returns an invalid code error. If the country is not found, it returns
// a not found error.
func Alpha2(code string) (*Country, error) {
	if len(code) != 2 {
		return nil, ErrInvalidCode
	}

	if code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
		return nil, ErrInvalidCode
	}

	country := alpha2Lookup[code[0]-'A'][code[1]-'A']
	if country == nil {
		return nil, ErrNotFound
	}
	return country, nil
}

// Fast lookup for a country by its Alpha-3 code. If the code is not three uppercase
// characters, it returns an invalid code error. If the country is not found, it returns
// a not found error.
func Alpha3(code string) (*Country, error) {
	if len(code) != 3 {
		return nil, ErrInvalidCode
	}
	if code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' || code[2] < 'A' || code[2] > 'Z' {
		return nil, ErrInvalidCode
	}

	country := alpha3Lookup[code[0]-'A'][code[1]-'A'][code[2]-'A']
	if country == nil {
		return nil, ErrNotFound
	}
	return country, nil
}

// Find a country by its name, including unofficial names. This function uses a trie
// structure to lookup the country by its various names. Finding an Alpha2 or Alpha3
// code is much faster using those specific functions. If the country is not found, it
// returns a false boolean instead of an error.
func Find(name string) (*Country, bool) {
	return root.Find(name)
}

// Flag returns the emoji flag representation of the country from the Alpha-2 code.
func Flag(code string) (string, error) {
	code = strings.ToUpper(code)
	if _, err := Alpha2(code); err != nil {
		return "", ErrInvalidCode
	}

	emoji := ""
	for _, r := range code {
		emoji += string(r + 0x1F1A5)
	}
	return emoji, nil
}

// Flag returns the emoji flag representation of the country.
func (c *Country) Flag() string {
	if c.flag == "" {
		c.flag, _ = Flag(c.Alpha2)
	}
	return c.flag
}
