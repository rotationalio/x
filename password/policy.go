package password

import (
	"encoding/json"
	"fmt"
	"strings"
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
