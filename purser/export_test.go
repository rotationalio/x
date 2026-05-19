package purser

// Test-only exports. Anything declared here is visible to internal tests in package
// purser and to external tests in package purser_test, but is invisible outside test
// builds.

// NewNilPurser returns a typed-nil *purserImpl wrapped in the Purser interface.
// External tests use this to exercise the nil-receiver contract on the unexported
// concrete type without exposing it in the production API.
func NewNilPurser() Purser {
	var p *purserImpl
	return p
}
