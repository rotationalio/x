package password

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"

	"go.rtnl.ai/x/randstr"
)

const (
	generationAttempts    = 8
	defaultPasswordLength = 14
)

var (
	ErrGenerationFailed = errors.New("failed to generate password that meets policy requirements")
)

type Policy struct {
	Strength  Strength          `json:"strength,omitempty"`        // minimum strength of the passwords to match by this policy
	Length    *Range            `json:"length,omitempty,omitzero"` // length of the password to generate
	Charsets  []*CharSelect     `json:"charsets,omitempty"`        // character sets for the policy
	Define    map[string]string `json:"define,omitempty"`          // define character sets for the policy
	Require   []string          `json:"require,omitempty"`         // require characters from these character sets in the password
	URLEncode bool              `json:"urlencode,omitempty"`       // URL encode the password
}

type Range struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

type CharSelect struct {
	Name string  `json:"name"` // name of the character set
	Prob float64 `json:"prob"` // probability of selecting the character set
}

// Charset returns the character set for the given name. If the character set is
// defined in the policy it is returned, otherwise the default character set is
// returned. If the name is unknown an empty string is returned.
func (p *Policy) Charset(name string) string {
	if cs, ok := p.Define[name]; ok {
		return cs
	}
	name = strings.ToLower(strings.TrimSpace(name))
	return charsets[name]
}

// Returns an error if the password does not match the policy, otherwise nil.
func (p *Policy) Check(password string) error {
	// If the strength is set ensure the password is at least that strong.
	if p.Strength > Insecure {
		if strength := Check(password); strength < p.Strength {
			return fmt.Errorf("password strength is %s, minimum required strength is %s", strength, p.Strength)
		}
	}

	// If the length is set ensure the password is at least that long.
	if p.Length != nil {
		if p.Length.Min > 0 && len(password) < int(p.Length.Min) {
			return fmt.Errorf("password length is %d, minimum required length is %d", len(password), p.Length.Min)
		}
		if p.Length.Max != 0 && p.Length.Max > p.Length.Min && len(password) > int(p.Length.Max) {
			return fmt.Errorf("password length is %d, maximum allowed length is %d", len(password), p.Length.Max)
		}
	}

	// Check that the password contains all the required character sets.
	for _, charset := range p.Require {
		if !Contains(password, p.Charset(charset)) {
			return fmt.Errorf("password does not contain required character set %s", charset)
		}
	}

	return nil
}

// Generates a password that matches the policy. Password generation attempts to at
// most 8 attempts to generate a password that matches the policy, if it cannot it
// returns an error.
func (p *Policy) Generate() (string, error) {
	// Determine the length of the password to generate.
	var len int
	if p.Length != nil {
		len = p.Length.Get()
	} else {
		len = defaultPasswordLength
	}

	for range generationAttempts {
		pw := randstr.Generate(len, Charset("alphasymbolic"))
		if err := p.Check(pw); err == nil {
			return pw, nil
		}
	}
	return "", ErrGenerationFailed
}

//============================================================================
// Password Generation Helpers
//============================================================================

func (r *Range) Get() int {
	if r.Min == r.Max {
		if r.Min == 0 {
			return defaultPasswordLength
		}
		return int(r.Min)
	}

	if r.Max > r.Min {
		return rand.Intn(int(r.Max-r.Min)+1) + int(r.Min)
	}

	return int(r.Min)
}

// Returns a character distribution with randomized probabilities that sum to 1.0 or
// integer values that define the minimum number of characters for each charset.
func NormalizeCharDist(dist []*CharSelect) []*CharSelect {
	if len(dist) == 0 {
		return defaultCharDist
	}

	// Check to ensure that the total probability of the charsets is 1.0
	//cSpell:ignore nzero, nints
	total := 0.0
	nzero := 0
	nints := 0
	for _, charset := range dist {
		total += charset.Prob
		if round(charset.Prob) == 0.0 {
			nzero++
		} else if charset.Prob == math.Trunc(charset.Prob) {
			nints++
		}
	}

	// If the distribution adds up to 1.0 then it is normalized
	total = round(total)
	if total == 1.0 {
		return dist
	}

	// if the distribution is all integers then return the distribution as is
	if nints+nzero == len(dist) {
		return dist
	}

	// If the total is less than 1.0 then distribute the rest of the
	// probability across the zero probability charsets.
	if nzero > 0 && total < 1.0 {
		remainder := 1.0 - total
		for _, charset := range dist {
			if round(charset.Prob) == 0.0 {
				charset.Prob = remainder / float64(nzero)
			}
		}
		return dist
	}

	// If the total is less than 1.0 and there are no zero probability
	// charsets then distribute the rest of the probability evenly across
	// the charsets.
	if nzero == 0 && total < 1.0 {
		incr := (1.0 - total) / float64(len(dist))
		for _, charset := range dist {
			charset.Prob += incr
		}
		return dist
	}

	// Pathological case: convert to integers and change 0 to 1
	for _, charset := range dist {
		charset.Prob = math.Trunc(charset.Prob)
		if charset.Prob == 0 {
			charset.Prob = 1
		}
	}
	return dist
}

var (
	defaultCharDist = []*CharSelect{
		{Name: "uppercase", Prob: 0.35},
		{Name: "lowercase", Prob: 0.35},
		{Name: "digits", Prob: 0.2},
		{Name: "symbols", Prob: 0.1},
	}
)

//============================================================================
// JSON Marshal and Unmarshal Custom Types
//============================================================================

func (r *Range) IsZero() bool {
	return r.Min == 0 && r.Max == 0
}

// Marshal the range as a single integer if the min and max are the same, otherwise
// marshal the range as a JSON object with min and max properties.
func (r *Range) MarshalJSON() ([]byte, error) {
	if r.Min == r.Max {
		return json.Marshal(r.Min)
	}

	rng := map[string]int64{
		"min": r.Min,
		"max": r.Max,
	}

	return json.Marshal(rng)
}

// Unmarshal the range from a single integer if the value is an integer, otherwise
// unmarshal the range from a JSON object with min and max properties.
func (r *Range) UnmarshalJSON(data []byte) error {
	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		if r.Min, err = num.Int64(); err != nil {
			return err
		}
		r.Max = r.Min
		return nil
	}

	var rng map[string]int64
	if err := json.Unmarshal(data, &rng); err == nil {
		r.Min = rng["min"]
		r.Max = rng["max"]
		return nil
	}

	return fmt.Errorf("could not unmarshal range")
}

func (c *CharSelect) IsZero() bool {
	return c.Name == "" && c.Prob == 0
}

func (c *CharSelect) MarshalJSON() ([]byte, error) {
	if c.Prob == 0 {
		return json.Marshal(c.Name)
	}

	cs := map[string]any{
		"name": c.Name,
		"prob": c.Prob,
	}

	return json.Marshal(cs)
}

func (c *CharSelect) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		c.Name = name
		return nil
	}

	var cs struct {
		Name string  `json:"name"`
		Prob float64 `json:"prob"`
	}

	if err := json.Unmarshal(data, &cs); err == nil {
		c.Name = cs.Name
		c.Prob = cs.Prob
		return nil
	}

	return fmt.Errorf("could not unmarshal charselect")
}

func round(f float64) float64 {
	ratio := math.Pow(10, 4)
	return math.Round(f*ratio) / ratio
}
