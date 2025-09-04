// Library ported and adapted from https://github.com/google/go-querystring
// Copyright (c) 2013 Google. All rights reserved.
package query

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Encoder is an interface implemented by any type that wishes to encode
// itself into URL values in a non-standard way.
type Encoder interface {
	EncodeValues(key string, v *url.Values) error
}

// Values returns the url.Values encoding of v.
//
// Values expects to be passed a struct, and traverses it recursively to encode each
// struct field as a URL parameter unless:
//
//   - the field's tag is "-", or
//   - the field is empty and its tag specifies the "omitempty" option.
//
// The empty values are false, 0, any nil pointer or interface value, and any array,
// slice, map, or string of zero length, and any type (such as time.Time) that returns
// true for IsZero().
//
// The URL parameter name defaults to the struct field name but can be specified via a
// tag value (recommended). The "url" key in the tag value is the key name, followed by
// an optional comma and options. Examples:
//
// // Field is ignored by this package
// Field int `url:"-"`
//
// // Field appears as URL parameter "myParam"
// Field int `url:"myParam"
//
// // Field appears as URL parameter "myParam", but only if non-empty
// Field int `url:"myParam,omitempty"`
//
// // Field appears as URL parameter "Field" (the default), but only if non-empty.
// // Note the leading comma required for options
// Field int `url:",omitempty"`
//
// For encoding individual field values, the following type-dependent rules apply:
//
// Bool values are encoded as "true" or "false" unless the "int" option is specified,
// in which case they are encoded as "1" or "0".
//
// time.Time values default to RFC3339 format. Including the "unix" option signals that
// the field should be encoded as a Unix time, and unixmilli and unixnano will encode
// the number of milliseconds and nanoseconds since the Unix epoch. Including the
// "layout" tag separate from the "url" tag specifies a custom time format layout.
//
// Slice and array values are encoded as multiple instances of the same parameter name.
// Including the "comma" option signals that the field should be encoded as a single
// parameter with comma-separated values. Including the "space" or "semicolon" options
// will similarly delimit the slices. The "brackets" option signals that multiple URL
// values should have "[]" appended to the value name. "numbered" will append a number to
// the end of each instance of the value name.
//
// Anonymous struct fields are treated as if their inner fields were part of the outer
// struct, subject to the usual rules. An anonymous struct with a name given in its
// URL tag is treated as having that name, rather than being anonymous.
//
// Non-nil pointer values are encoded as the value pointed to.
//
// Nested structs have their fields processed recursively and are encoded including the
// parent fields in the value names for scoping. For example:
//
// "user[name]=acme&user[addr][postcode]=12345&user[addr][city]=Metropolis"
//
// All other values are encoded using their default string representation.
//
// Multiple fields that encode to the same URL parameter name will be included as
// multiple URL values of the same name.
func Values(v any) (url.Values, error) {
	values := make(url.Values)
	if v == nil {
		return values, nil
	}

	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return values, nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("query: Values() expects struct input. Got %v", val.Kind())
	}

	err := reflectValue(values, val, "")
	return values, err
}

var (
	timeType    = reflect.TypeOf(time.Time{})
	encoderType = reflect.TypeOf(new(Encoder)).Elem()
)

func reflectValue(values url.Values, val reflect.Value, scope string) error {
	var embedded []reflect.Value

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous { // unexported
			continue
		}

		sv := val.Field(i)
		tag := sf.Tag.Get("url")
		if tag == "-" {
			continue
		}
		name, opts := parseTag(tag)

		if name == "" {
			if sf.Anonymous {
				v := reflect.Indirect(sv)
				if v.IsValid() && v.Kind() == reflect.Struct {
					embedded = append(embedded, v)
					continue
				}
			}
			name = sf.Name
		}

		if scope != "" {
			name = scope + "[" + name + "]"
		}

		if opts.Contains("omitempty") && isEmptyValue(sv) {
			continue
		}

		if sv.Type().Implements(encoderType) {
			// if sv is a nil pointer and the custom encoder is defined on a non-pointer
			// method receiver, set sv to the zero value of the underlying type.
			if !reflect.Indirect(sv).IsValid() && sv.Type().Elem().Implements(encoderType) {
				sv = reflect.New(sv.Type().Elem())
			}

			m := sv.Interface().(Encoder)
			if err := m.EncodeValues(name, &values); err != nil {
				return err
			}
			continue
		}

		// recursively dereference pointers. break on nil pointers
		for sv.Kind() == reflect.Ptr {
			if sv.IsNil() {
				break
			}
			sv = sv.Elem()
		}

		if sv.Kind() == reflect.Slice || sv.Kind() == reflect.Array {
			if sv.Len() == 0 {
				// empty slice, skip
				continue
			}

			var delim string
			if opts.Contains("comma") {
				delim = ","
			} else if opts.Contains("space") {
				delim = " "
			} else if opts.Contains("semicolon") {
				delim = ";"
			} else if opts.Contains("brackets") {
				name += "[]"
			} else {
				delim = sf.Tag.Get("del")
			}

			if delim != "" {
				sb := new(strings.Builder)
				first := true
				for i := 0; i < sv.Len(); i++ {
					if first {
						first = false
					} else {
						sb.WriteString(delim)
					}
					sb.WriteString(valueRepr(sv.Index(i), opts, sf))
				}
				values.Add(name, sb.String())
			} else {
				for i := 0; i < sv.Len(); i++ {
					k := name
					if opts.Contains("numbered") {
						k = fmt.Sprintf("%s%d", name, i)
					}
					values.Add(k, valueRepr(sv.Index(i), opts, sf))
				}
			}
			continue
		}

		if sv.Type() == timeType {
			values.Add(name, valueRepr(sv, opts, sf))
			continue
		}

		if sv.Kind() == reflect.Struct {
			if err := reflectValue(values, sv, name); err != nil {
				return err
			}
			continue
		}

		values.Add(name, valueRepr(sv, opts, sf))
	}

	for _, f := range embedded {
		if err := reflectValue(values, f, scope); err != nil {
			return err
		}
	}

	return nil
}

func valueRepr(v reflect.Value, opts tagOptions, sf reflect.StructField) string {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Bool && opts.Contains("int") {
		if v.Bool() {
			return "1"
		}
		return "0"
	}

	if v.Type() == timeType {
		t := v.Interface().(time.Time)
		if opts.Contains("unix") {
			return strconv.FormatInt(t.Unix(), 10)
		}
		if opts.Contains("unixmilli") {
			return strconv.FormatInt(t.UnixMilli(), 10)
		}
		if opts.Contains("unixnano") {
			return strconv.FormatInt(t.UnixNano(), 10)
		}
		if layout := sf.Tag.Get("layout"); layout != "" {
			return t.Format(layout)
		}
		return t.Format(time.RFC3339)
	}

	return fmt.Sprint(v.Interface())
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}

	type zeroable interface {
		IsZero() bool
	}

	if z, ok := v.Interface().(zeroable); ok {
		return z.IsZero()
	}

	return false
}

//===========================================================================
// Tag Parsing
//===========================================================================

// tagOptions is the string following a comma in a struct field's "url" tag,
// or the empty string. It does not include the leading comma.
type tagOptions []string

func parseTag(tag string) (string, tagOptions) {
	s := strings.Split(tag, ",")
	return s[0], s[1:]
}

func (o tagOptions) Contains(opt string) bool {
	for _, option := range o {
		if option == opt {
			return true
		}
	}
	return false
}
