package quant

// ############################################################################
// Stemmer
// ############################################################################

// TODO: docs
type Stemmer struct {
	stemmer StemmerImpl //TODO StemmerOption func
	lang    Language    //TODO StemmerOption func
}

// StemmerImpl is the type of a function that implements a steming algorithm for
// use by a [Stemmer].
type StemmerImpl func(s *Stemmer, token string) (stem string)

// TODO: docs and defaults
func NewStemmer() *Stemmer {
	return &Stemmer{
		stemmer: PorterStemmer,
	}
}

// Stem returns a stem for the given token. //TODO: list default options here
func (s *Stemmer) Stem(token string, opts ...StemmerOption) (stem string) {
	return s.stemmer(s, token)
}

// ############################################################################
// StemmerOptions
// ############################################################################

// TODO: docs
type StemmerOption func(s *Stemmer)

// ############################################################################
// PorterStemmer
// ############################################################################

// TODO docs
func PorterStemmer(s *Stemmer, token string) (stem string) {
	// Porter stemmer only supports English
	//FIXME: error or something if not english?
	if !s.lang.In(LanuageEnglish) {
		return token
	}

	//TODO: implement porter stemmer
	return stem
}
