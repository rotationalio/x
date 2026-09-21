package password

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.rtnl.ai/x/randstr"
)

const (
	generationAttempts    = 8
	defaultPasswordLength = 14
	configEnvVar          = "PASSWORD_POLICIES"
)

var (
	ErrGenerationFailed = errors.New("failed to generate password that meets policy requirements")
)

var (
	basicPolicy = &Policy{
		Length: &Range{
			Min: 9,
			Max: 16,
		},
		Charsets: []*CharSelect{
			{Name: "differentiable"},
		},
	}
	strongPolicy = &Policy{
		Length: &Range{
			Min: 16,
			Max: 16,
		},
		Charsets: []*CharSelect{
			{Name: "uppercase", Prob: 0.35},
			{Name: "lowercase", Prob: 0.35},
			{Name: "digits", Prob: 0.2},
			{Name: "symbols", Prob: 0.1},
		},
		Require: []string{"uppercase", "lowercase", "digits", "symbols"},
	}
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

// Loads a policy based on the specified name.
func Load(name string) (_ *Policy, err error) {
	var policies map[string]*Policy
	if policies, err = LoadAll(); err != nil {
		return nil, err
	}

	if name == "" {
		name = "default"
	}

	if policy, ok := policies[name]; ok {
		return policy, nil
	}

	return nil, fmt.Errorf("unknown or undefined policy %q", name)
}

func ConfigPath() string {
	var configPath string
	if configPath = os.Getenv(configEnvVar); configPath == "" {
		if home, _ := os.UserHomeDir(); home != "" {
			configPath = filepath.Join(home, ".config", "mkpasswd", "policies.json")
		}
	}

	// Expand the environment variables in the config path.
	configPath = os.ExpandEnv(configPath)
	return configPath
}

// Loads all policies from the configuration, defaulting to the basic and strong
// policies (and including the default policy if specified).
func LoadAll() (policies map[string]*Policy, err error) {
	if configPath := ConfigPath(); configPath != "" {
		if policies, err = loadPolicies(configPath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("could not load policies from %s: %w", configPath, err)
			}
		}
	}

	if policies == nil {
		policies = make(map[string]*Policy)
	}

	// Register the built in policies
	if _, ok := policies["basic"]; !ok {
		policies["basic"] = basicPolicy
	}

	if _, ok := policies["default"]; !ok {
		policies["default"] = strongPolicy
	}

	return policies, nil
}

func loadPolicies(path string) (policies map[string]*Policy, err error) {
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return nil, err
	}
	defer f.Close()

	policies = make(map[string]*Policy)
	if err = json.NewDecoder(f).Decode(&policies); err != nil {
		return nil, err
	}
	return policies, nil
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
func (p *Policy) Check(password string) (err error) {
	// If the policy requires URL encoding then decode the password.
	if p.URLEncode {
		if password, err = url.PathUnescape(password); err != nil {
			return fmt.Errorf("failed to unescape password: %w", err)
		}
	}

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
	// Ensure that all of the character sets are defined.
	for _, ch := range p.Charsets {
		if charset := p.Charset(ch.Name); charset == "" {
			return "", fmt.Errorf("unknown or undefined character set %q is not defined", ch.Name)
		}
	}

	// Ensure that all of the required character sets are in the character sets.
	for _, name := range p.Require {
		if charset := p.Charset(name); charset == "" {
			return "", fmt.Errorf("required character set %q is not defined", name)
		}
	}

	// Normalize all of the character sets.
	dist := NormalizeCharDist(p.Charsets)

attempts:
	for range generationAttempts {
		// Determine the length of the password to generate.
		var len int
		if p.Length != nil {
			len = p.Length.Get()
		} else {
			len = defaultPasswordLength
		}

		if len == 0 {
			// If the length is not set then skip this attempt.
			continue attempts
		}

		// Generate the counts for each of the character sets.
		counts := CharsetCounts(len, dist)
		for _, name := range p.Require {
			if counts[name] == 0 {
				// If the required character set is not represented then skip this attempt.
				continue attempts
			}
		}

		pw := ""
		for name, count := range counts {
			charset := p.Charset(name)
			pw += randstr.Generate(count, charset)
		}

		// Randomly shuffle the characters.
		pw = Shuffle(pw)

		if err := p.Check(pw); err == nil {
			// URLEncode the password if required
			if p.URLEncode {
				return url.PathEscape(pw), nil
			}
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

// Returns the number of characters for each charset in the distribution.
func CharsetCounts(n int, dist []*CharSelect) map[string]int {
	dist = NormalizeCharDist(dist)

	probs := 0.0
	for _, charset := range dist {
		probs += charset.Prob
	}

	// Two options: integers or probabilities.
	probs = round(probs)
	if probs == 1.0 {
		// Use probabilities
		return probabilityDistribution(n, dist)
	}

	// Otherwise use integers.
	return integerDistribution(n, dist)
}

// Performs roulette wheel selection to distribute the characters across the charsets.
func probabilityDistribution(n int, dist []*CharSelect) map[string]int {
	counts := make(map[string]int)
	cumulative := make([]float64, len(dist))
	for i, charset := range dist {
		if i == 0 {
			cumulative[i] = charset.Prob
		} else {
			cumulative[i] = cumulative[i-1] + charset.Prob
		}
	}

	for range n {
		rand := rand.Float64()
		for i, cum := range cumulative {
			if rand < cum {
				counts[dist[i].Name]++
				break
			}
		}
	}
	return counts
}

// Distributes the characters across the charsets using the counts as weighted suggestions.
func integerDistribution(n int, dist []*CharSelect) map[string]int {
	remaining := n

	counts := make(map[string]int)
	unused := make([]string, 0, len(dist))

	// Allocate the characters to the charsets based on the counts.
	for _, charset := range dist {
		count := int(math.Trunc(charset.Prob))
		if count == 0 {
			unused = append(unused, charset.Name)
			continue
		}

		if count > remaining {
			count = remaining
		}

		counts[charset.Name] = count
		remaining -= count

		if remaining == 0 {
			return counts
		}
	}

	if remaining > 0 {
		if len(unused) > 0 {
			for remaining > 0 {
				for _, charset := range unused {
					counts[charset]++
					remaining--
					if remaining == 0 {
						return counts
					}
				}
			}
		} else {
			for remaining > 0 {
				for _, charset := range dist {
					counts[charset.Name]++
					remaining--
					if remaining == 0 {
						return counts
					}
				}
			}
		}
	}

	// Should never make it here.
	return counts
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
