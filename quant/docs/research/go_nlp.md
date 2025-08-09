# Go NLP Notes

## Overall Thoughts

* I haven't found anything super-exhaustive like NLTK or spaCy or things like that which have everything you could want in NLP under one roof; most of the Go stuff is smaller tools with specific purposes
* There is plenty of MIT licensed Go code that does stuff like stemming, tokenization, etc that we could use or gain inspiration from (see "Go NLP Libraries" (some of it has poor developer practices, some of it looks good enough, but nothing looks as good as Rotational written code)
* NLP libraries can be anything from stemmers to tokenizers to model trainers to etc., so searching "NLP libraries go" is useless, I should search "tokenizer Go" or "stemmer go" or even "porter stemmer go" to be more effective
* Awesome lists are still great for finding tools and code for specific topics, but aren't exhaustive
* Most of the libraries use []rune or []byte for performance
* It's possible to do cgo wrappers for compiled libraries, so if there's a SOTA C lib that does certain things I could wrap it for our use, however that also goes into licensing and it's not a free task, it's still a bit of work and I've never done it and there's a cost in performance as far as I know

## "Awesome NLP" Lists

* This one is smaller but has pretty good libraries: <https://github.com/gopherdata/resources/blob/master/tooling/README.md#nlp>
* This one is large and has a lot of garbage on it but some gems: <https://awesome-go.com/natural-language-processing/>

## Go NLP libraries that I've looked at enough to comment on

Function | Link | License | Comments
-|-|-|-
porter stemmer|<https://github.com/reiver/go-porterstemmer>| MIT :cookie:|\* A go native porter stemmer \* Uses []rune to make things more efficient
porter stemmer|<https://github.com/agonopol/go-stem>|MIT  :cookie:|\* A go native stemmer \* Uses []byte for inputs and outputs and for efficiency \* Single-file implementation that looks very simple
porter stemmer|<https://github.com/a2800276/porter/blob/master/stemmer.go>|MIT :cookie:|\* A go native porter stemmer \* Single file \* Uses []byte
porter stemmer|<https://github.com/kampsy/gwizo>|custom> license :alien: :x:|\* A go native porter stemmer \* Very simple, uses strings \* Has a custom license (not standard)
porter2/snowball stemmer|<https://github.com/kljensen/snowball>|MIT> :cookie:|\* A go native porter2/snowball stemmer \* Many languages supported: English, Spanish (español), French (le français), Russian (ру́сский язы́к), Swedish (svenska), Norwegian (norsk), Hungarian (magyar) \* Uses []rune to make things more efficient
porter2/snowball stemmer|<<https://github.com/zentures/porter2>|Apache> 2.0 :+1:|\* A go native porter2/snowball stemmer \* Uses finite state machines to be super fast \* I probably can't review the code since it's a 2000 line Go finite state machine, but the author used a tool to generate a lot of the code, which may be reviewable \* This might be useful if we really want a fast porter2 stemmer, otherwise it's not worth looking at other than considering we might use FSMs for some of the text processing
porter and porter2/snowball stemmer|<https://github.com/goodsign/snowball> (wraps: <https://github.com/zvelo/libstemmer>)|BSD> 2-clause :+1:|\* Wrapper for libstemmer C library which does porter and porter2/snowball stemming \* It's a wrapper, so it might be useful to look at if we want to do our own wrapper for C code someday
porter and porter2/snowball stemmer|<https://github.com/rjohnsondev/golibstemmer> (wraps: <https://github.com/zvelo/libstemmer>)|none>, possibly BSD 2 clause? :question: :x:|\* Wrapper for libstemmer C library which does porter and porter2/snowball stemming \* No license! (covered BSD 2-clause because that's what libstemmer uses?) \* It's a wrapper, so it might be useful to look at if we want to do our own wrapper for C code someday
tokenize/segment/tag|<https://github.com/jdkato/prose>|MIT :cookie:|\* A go native library to do tokenizing, segmenting, and tagging \* The repo is frozen (not a bad thing at all if it works fine) \* It uses the options function pattern that Patrick suggested I use where it takes a variadic function type which modify the options on a struct (is this pretty much the builder pattern?) \* User inputs a string into a Document to do the processing (can customize which processing occurs) \* It has good accuracy and speed, according to the benchmarks in the repository \* You access the tokens/entities/segments using a range iterator on a function that returns those items
embeddings|<https://github.com/ynqa/wego>|Apache> 2.0 :+1:|\* A go native library for embeddings based on models \* Supports: Word2Vec, GloVe, and LexVec \* Follows good Go project structure, unlike a LOT of other NLP Go libraries
