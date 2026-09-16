package password

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Policy struct {
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
