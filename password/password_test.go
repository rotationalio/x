package password_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"

	. "go.rtnl.ai/x/password"
)

// cSpell:disable
func TestCheck(t *testing.T) {
	tests := []struct {
		password string
		strength Strength
	}{
		{password: "1f2!Ga5", strength: Insecure},
		{password: "password", strength: Insecure},
		{password: "theeaglefliesatmidnight", strength: Weak},
		{password: "apple cookie banker mediocre follows grease format plaster", strength: Soft},
		{password: "Franklin1234", strength: Moderate},
		{password: "Franklin1234!", strength: Hard},
		{password: "Appl3 Cook1e B4nker Med1ocr3 FoLlows GreaSe F0rm4t Pl4st3r", strength: Strong},
		{password: "cMZr2lHA-LeE~J8wpy.c", strength: Robust},
		{password: "KcGPZ2f9.nXN1Q7b9EzA36NaQKR+D~v4", strength: Durable},
		{password: "KcGPZ2f9.nXNIQ7b9EzA36NaQKR+D~v4", strength: Robust},
		{password: "KcGPZ2f9.nXNIQ7b9EzA36NaQKR+++D~v4", strength: Strong},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("Password %d", i), func(t *testing.T) {
			s := Check(tc.password)
			if s != tc.strength {
				// Print the analysis of how the strength was calculated.
				a := Analyze(tc.password)
				assert.Equal(t, a, s, "test %d: analyze output must match check output", i)
			}
			assert.Equal(t, s, tc.strength, "test %d: want %s, got %s", i, tc.strength.String(), s.String())
		})
	}
}

func TestAnalyze(t *testing.T) {
	tests := []string{
		"1f2!Ga5",
		"password",
		"theeaglefliesatmidnight",
		"apple cookie banker mediocre follows grease format plaster",
		"Franklin1234",
		"Franklin1234!",
		"Appl3 Cook1e B4nker Med1ocr3 FoLlows GreaSe F0rm4t Pl4st3r",
		"cMZr2lHA-LeE~J8wpy.c",
		"KcGPZ2f9.nXN1Q7b9EzA36NaQKR+D~v4",
		"KcGPZ2f9.nXNIQ7b9EzA36NaQKR+D~v4",
		"KcGPZ2f9.nXNIQ7b9EzA36NaQKR+++D~v4",
	}

	for i, tc := range tests {
		assert.Equal(t, Analyze(tc), Check(tc), "test %d: analyze output must match check output", i)
	}
}

//cSpell:enable

func TestStrength_JSON(t *testing.T) {
	tests := []Strength{Insecure, Weak, Soft, Moderate, Hard, Strong, Robust, Durable}
	for _, test := range tests {
		t.Run(test.String(), func(t *testing.T) {
			data, err := json.Marshal(test)
			assert.Ok(t, err)

			var v Strength
			err = json.Unmarshal(data, &v)
			assert.Ok(t, err)
			assert.Equal(t, v, test)
		})
	}

	t.Run("Invalid", func(t *testing.T) {
		var v Strength
		err := json.Unmarshal([]byte(`"invalid"`), &v)
		assert.Error(t, err)
		assert.Equal(t, v, Insecure)
	})
}

func TestStrength_Incr(t *testing.T) {
	assert.Equal(t, Insecure.Incr(), Weak)
	assert.Equal(t, Weak.Incr(), Soft)
	assert.Equal(t, Soft.Incr(), Moderate)
	assert.Equal(t, Moderate.Incr(), Hard)
	assert.Equal(t, Hard.Incr(), Strong)
	assert.Equal(t, Strong.Incr(), Robust)
	assert.Equal(t, Robust.Incr(), Durable)
	assert.Equal(t, Durable.Incr(), Durable)
}

func TestStrength_Decr(t *testing.T) {
	assert.Equal(t, Insecure.Decr(), Insecure)
	assert.Equal(t, Weak.Decr(), Insecure)
	assert.Equal(t, Soft.Decr(), Weak)
	assert.Equal(t, Moderate.Decr(), Soft)
	assert.Equal(t, Hard.Decr(), Moderate)
	assert.Equal(t, Strong.Decr(), Hard)
	assert.Equal(t, Robust.Decr(), Strong)
	assert.Equal(t, Durable.Decr(), Robust)
}

func TestStrength_Add(t *testing.T) {
	t.Run("Floor", func(t *testing.T) {
		assert.Equal(t, Insecure.Add(-1), Insecure)
		assert.Equal(t, Weak.Add(-2), Insecure)
		assert.Equal(t, Soft.Add(-3), Insecure)
		assert.Equal(t, Moderate.Add(-4), Insecure)
		assert.Equal(t, Hard.Add(-5), Insecure)
		assert.Equal(t, Strong.Add(-6), Insecure)
		assert.Equal(t, Robust.Add(-7), Insecure)
		assert.Equal(t, Durable.Add(-8), Insecure)
	})

	t.Run("Ceiling", func(t *testing.T) {
		assert.Equal(t, Insecure.Add(8), Durable)
		assert.Equal(t, Weak.Add(7), Durable)
		assert.Equal(t, Soft.Add(6), Durable)
		assert.Equal(t, Moderate.Add(5), Durable)
		assert.Equal(t, Hard.Add(4), Durable)
		assert.Equal(t, Strong.Add(3), Durable)
		assert.Equal(t, Robust.Add(2), Durable)
		assert.Equal(t, Durable.Add(1), Durable)
	})

	t.Run("Normal", func(t *testing.T) {
		assert.Equal(t, Insecure.Add(3), Moderate)
		assert.Equal(t, Weak.Add(3), Hard)
		assert.Equal(t, Soft.Add(3), Strong)
		assert.Equal(t, Moderate.Add(3), Robust)
		assert.Equal(t, Hard.Add(3), Durable)
		assert.Equal(t, Strong.Add(-3), Soft)
		assert.Equal(t, Robust.Add(-3), Moderate)
		assert.Equal(t, Durable.Add(-3), Hard)
	})
}
