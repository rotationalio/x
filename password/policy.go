package password

type Policy struct {
	Define map[string]string
}

// Charset returns the character set for the given name. If the character set is
// defined in the policy it is returned, otherwise the default character set is
// returned. If the name is unknown an empty string is returned.
func (p *Policy) Charset(name string) string {
	if cs, ok := p.Define[name]; ok {
		return cs
	}
	return charsets[name]
}
