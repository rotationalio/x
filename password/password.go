package password

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	uppercase      = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase      = "abcdefghijklmnopqrstuvwxyz"
	numbers        = "1234567890"
	symbols        = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~ "
	differentiable = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRTUVWXY1234567890"
	alphanumeric   = uppercase + lowercase + numbers
	alphasymbolic  = alphanumeric + symbols
)

// Default character sets defined by name
var charsets = map[string]string{
	"uppercase":      uppercase,
	"upper":          uppercase,
	"lowercase":      lowercase,
	"lower":          lowercase,
	"numbers":        numbers,
	"digits":         numbers,
	"symbols":        symbols,
	"differentiable": differentiable,
	"alphanumeric":   alphanumeric,
	"alphasymbolic":  alphasymbolic,
}

// Returns the strength of the password as computed by the strength scoring algorithm.
func Check(password string) Strength {
	// Disqualify passwords that are too short or are dictionary words.
	if len(password) < 8 || IsDictionaryWord(password) {
		return Insecure
	}

	// Start with a base strength of 0
	var strength Strength
	switch {
	case len(password) >= 32:
		strength = strength.Add(3)
	case len(password) >= 16:
		strength = strength.Add(2)
	case len(password) > 8:
		strength = strength.Incr()
	}

	charsets := [4]string{
		charsets["uppercase"],
		charsets["lowercase"],
		charsets["numbers"],
		charsets["symbols"],
	}
	containsset := [4]bool{false, false, false, false}
	setsequence := []int{-1, -1, -1, -1}

	// Loop over the password only a single time for charset checks and duplication.
	for i, char := range password {
		for j, set := range charsets {
			if hasset := strings.ContainsRune(set, char); hasset {
				// Mark the charset as present in the password
				containsset[j] = hasset

				// Update the sequence of the charset.
				setsequence = append(setsequence[1:4], j)

				// No need to check the other charsets.
				break
			}
		}

		// Check for the duplication of the charset 4 times in a row.
		if i > 3 {
			if setsequence[0] == setsequence[1] && setsequence[0] == setsequence[2] && setsequence[0] == setsequence[3] {
				strength = strength.Decr()
			}
		}

		// Decrement the strength for duplication of the previous character.
		if i > 1 && password[i] == password[i-1] && password[i] == password[i-2] {
			strength = strength.Decr()
		}
	}

	// Increment the strength for each charset that is present.
	for _, found := range containsset {
		if found {
			strength = strength.Incr()
		}
	}
	return strength
}

func Analyze(password string) Strength {
	// Disqualify passwords that are too short or are dictionary words.
	if len(password) < 8 {
		fmt.Printf("%q is less than 8 characters\n", password)
		return Insecure
	}

	// Disqualify passwords that are dictionary words.
	// NOTE: this doesn't check if it contains a dictionary word as a substring.
	if IsDictionaryWord(password) {
		fmt.Printf("%q is a dictionary word\n", password)
		return Insecure
	}

	// Start with a base strength of 0
	var strength Strength
	switch {
	case len(password) >= 32:
		strength = strength.Add(3)
		fmt.Printf("more than 32 characters strength is now %s\n", strength.String())
	case len(password) >= 16:
		strength = strength.Add(2)
		fmt.Printf("more than 16 characters strength is now %s\n", strength.String())
	case len(password) > 8:
		strength = strength.Incr()
		fmt.Printf("more than 8 characters strength is now %s\n", strength.String())
	}

	charsets := [4]string{
		charsets["uppercase"],
		charsets["lowercase"],
		charsets["numbers"],
		charsets["symbols"],
	}
	containsset := [4]bool{false, false, false, false}
	setsequence := []int{-1, -1, -1, -1}

	// Loop over the password only a single time for charset checks and duplication.
	for i, char := range password {
		for j, set := range charsets {
			if hasset := strings.ContainsRune(set, char); hasset {
				// Mark the charset as present in the password
				containsset[j] = hasset

				// Update the sequence of the charset.
				setsequence = append(setsequence[1:4], j)

				// No need to check the other charsets.
				break
			}
		}

		// Check for the duplication of the charset 4 times in a row.
		if i > 3 {
			if setsequence[0] == setsequence[1] && setsequence[0] == setsequence[2] && setsequence[0] == setsequence[3] {
				strength = strength.Decr()
				switch setsequence[0] {
				case 0:
					fmt.Printf("uppercase charset is duplicated 4 times in a row (%v), strength is now %s\n", setsequence, strength.String())
				case 1:
					fmt.Printf("lowercase charset is duplicated 4 times in a row (%v), strength is now %s\n", setsequence, strength.String())
				case 2:
					fmt.Printf("numbers charset is duplicated 4 times in a row (%v), strength is now %s\n", setsequence, strength.String())
				case 3:
					fmt.Printf("symbols charset is duplicated 4 times in a row (%v), strength is now %s\n", setsequence, strength.String())
				}
			}
		}

		// Decrement the strength for duplication of the previous character.
		if i > 1 && password[i] == password[i-1] && password[i] == password[i-2] {
			strength = strength.Decr()
			fmt.Printf("character duplication %q, strength is now %s\n", string(password[i-2:i+1]), strength.String())
		}
	}

	// Increment the strength for each charset that is present.
	for i, found := range containsset {
		if found {
			strength = strength.Incr()
			switch i {
			case 0:
				fmt.Printf("uppercase charset is present, strength is now %s\n", strength.String())
			case 1:
				fmt.Printf("lowercase charset is present, strength is now %s\n", strength.String())
			case 2:
				fmt.Printf("numbers charset is present, strength is now %s\n", strength.String())
			case 3:
				fmt.Printf("symbols charset is present, strength is now %s\n", strength.String())
			}
		}
	}
	return strength
}

type Strength uint8

const (
	Insecure Strength = iota
	Weak
	Soft
	Moderate
	Hard
	Strong
	Robust
	Durable
)

var strengthStrings = [8]string{
	"insecure", "weak", "soft", "moderate", "hard", "strong", "robust", "durable",
}

// Strength returns the strength of the password.
func (s Strength) String() string {
	if s > Durable {
		return "unknown"
	}
	return strengthStrings[s]
}

func (s Strength) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Strength) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	str = strings.ToLower(strings.TrimSpace(str))
	for i, name := range strengthStrings {
		if name == str {
			*s = Strength(i)
			return nil
		}
	}

	return fmt.Errorf("invalid strength name: %q", str)
}

func (s Strength) Incr() Strength {
	if s < Durable {
		return Strength(int(s) + 1)
	}
	return s
}

func (s Strength) Decr() Strength {
	if s > Insecure {
		return Strength(int(s) - 1)
	}
	return s
}

func (s Strength) Add(n int) Strength {
	t := int(s) + n

	if t < 0 {
		return Insecure
	}

	if t > int(Durable) {
		return Durable
	}

	return Strength(t)
}
