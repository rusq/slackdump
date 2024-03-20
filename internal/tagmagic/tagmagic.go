package tagmagic

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	jsonTag = "json" // default struct tag to use
	defsep  = ","    // default tag separator
)

// JSONToMap converts data to a map.  Implementation supports embedded
// anonymous structures.
func JSONToMap(data any) map[string]any {
	m, _ := toMap(data, jsonTag, defsep, false)
	return m
}

// ToMap converts data to a map{"tag" : value}.  If `omitempty` is
// specified, then fields having empty values and tag option 'omitempty' are
// omitted (same behaviour as in encoding/json library).
func ToMap(data any, omitempty bool) map[string]any {
	m, _ := toMap(data, jsonTag, defsep, omitempty)
	return m
}

// ToMapWithTag converts data to a map{"tag" : value}, sep is the tag separator.
// For JSON tags, tag="json", sep=",".  If `omitempty` is specified, then fields
// having empty values and tag option 'omitempty' are omitted.
func ToMapWithTag(data any, tag string, sep string, omitempty bool) map[string]any {
	m, _ := toMap(data, tag, sep, omitempty)
	return m
}

// toMap converts a record to a map.  Implementation supports embedded
// anonymous structures.  It returns a map and the order of fields in the
// structure.
func toMap(data any, tag string, tagSep string, omitempty bool) (map[string]any, []string) {
	out := make(map[string]any)
	order := make([]string, 0)

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := typ.Field(i)

		if field.Type.Kind() == reflect.Struct && field.Anonymous {
			nested, norder := toMap(v.Field(i).Interface(), tag, tagSep, omitempty)
			for key, val := range nested {
				out[key] = val
			}
			order = append(order, norder...)
		} else {
			if !isExported(field.Name) {
				continue
			}
			tagValue := strings.SplitN(field.Tag.Get(tag), tagSep, 2)
			if tagValue[0] == "-" {
				continue
			}
			if tagValue[0] == "" {
				tagValue[0] = field.Name
			}
			if omitempty {
				// if there's a tag option and that tagoption is omitempty
				// and field is empty.
				if len(tagValue) > 1 && (tagValue[1] == "omitempty" && isEmptyValue(v.Field(i))) {
					continue
				}
			}

			out[tagValue[0]] = v.Field(i).Interface()
			order = append(order, tagValue[0])
		}
	}
	return out, order
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
	return false
}

// isExported returns true if the field is exported.
func isExported(fieldName string) bool {
	firstRune, _ := utf8.DecodeRuneInString(fieldName)
	if firstRune == utf8.RuneError {
		panic(fmt.Sprintf("isUnexported(): is that even a field: %s ", fieldName))
	}
	if unicode.In(firstRune, unicode.Lu) {
		return true
	}
	return false
}

// ColumnNames returns a sorted list of names for the fieldMap.
func ColumnNames(columnMap map[string]any) []string {
	fields := make([]string, 0, len(columnMap))
	for k := range columnMap {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	return fields
}

// ColumnValues populates `out` with values from `columnMap` in `columnOrder`
// order.  The size of `out` will be adjusted to columnOrder size to accomodate
// for all values.  This function is useful when populating binds.
// `columnMap` is a map with name of the column as a key, and value of the
// column as a value.
func ColumnValues(out *[]any, columnMap map[string]any, columnOrder []string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	if out == nil {
		return errors.New("nil pointer passed for out")
	}

	if len(*out) != len(columnOrder) {
		resize(out, len(columnOrder))
	}

	for i := range columnOrder {
		(*out)[i] = columnMap[columnOrder[i]]
	}
	return nil
}

// ExtractColumnNames returns a sorted list of column names given a struct
// object.  It uses the default "json" struct tag, and does not omit the empty
// fields.
func ExtractColumnNames(data any) []string {
	_, order := toMap(data, jsonTag, defsep, false)
	return order
}

// ExtractColumnNamesTag returns a sorted list of column names given a struct
// object.
func ExtractColumnNamesTag(data any, tag string, sep string, omitempty bool) []string {
	_, order := toMap(data, tag, sep, omitempty)
	return order
}

// resize resizes the slice to a requested size.  If slice is smaller - it
// is extended, if bigger - truncated to the desired size.
// Panics if s is nil.
func resize[T any](s *[]T, sz int) {
	if s == nil {
		panic("resize: nil input pointer")
	}
	if len(*s) >= sz {
		// chop off the tail
		*s = (*s)[:sz]
		return
	}
	// enlarge required
	add := make([]T, sz-len(*s))
	*s = append(*s, add...)
}
