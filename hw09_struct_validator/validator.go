package hw09structvalidator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	// Ошибки валидации.
	ErrInvalidLength = errors.New("invalid length")
	ErrInvalidRegexp = errors.New("invalid regexp")
	ErrInvalidIn     = errors.New("value is not in allowed set")
	ErrInvalidMin    = errors.New("value is less than minimum")
	ErrInvalidMax    = errors.New("value is greater than maximum")

	// Программные ошибки.
	ErrInvalidTag      = errors.New("invalid validate tag")
	ErrUnsupportedType = errors.New("unsupported type")
	ErrNotStruct       = errors.New("value is not a struct")
)

type ValidationError struct {
	Field string
	Err   error
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %v", e.Field, e.Err)
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}

	var b strings.Builder
	for i, e := range v {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(e.Error())
	}
	return b.String()
}

// Is позволяет использовать errors.Is(err, ErrInvalidLength) и т.п.
func (v ValidationErrors) Is(target error) bool {
	for _, e := range v {
		if errors.Is(e.Err, target) {
			return true
		}
	}
	return false
}

type rule struct {
	name string
	arg  string
}

func Validate(v interface{}) error {
	if v == nil {
		return ErrNotStruct
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ErrNotStruct
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	errs, err := validateStruct(rv)
	if err != nil {
		return err
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func validateStruct(rv reflect.Value) (ValidationErrors, error) {
	rt := rv.Type()
	var errs ValidationErrors

	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)

		// Валидируем только публичные поля.
		if sf.PkgPath != "" {
			continue
		}

		tag, ok := sf.Tag.Lookup("validate")
		if !ok || tag == "" {
			continue
		}

		rules, err := parseRules(tag)
		if err != nil {
			return nil, err
		}

		fv := rv.Field(i)

		var normalRules []rule
		nested := false

		for _, r := range rules {
			if r.name == "nested" {
				nested = true
			} else {
				normalRules = append(normalRules, r)
			}
		}

		if nested {
			nestedErrs, err := validateNested(sf.Name, fv)
			if err != nil {
				return nil, err
			}
			errs = append(errs, nestedErrs...)
		}

		if len(normalRules) > 0 {
			fieldErrs, err := validateValue(sf.Name, fv, normalRules)
			if err != nil {
				return nil, err
			}
			errs = append(errs, fieldErrs...)
		}
	}

	return errs, nil
}

func parseRules(tag string) ([]rule, error) {
	parts := strings.Split(tag, "|")
	rules := make([]rule, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("%w: empty rule", ErrInvalidTag)
		}

		name, arg, hasArg := strings.Cut(p, ":")
		name = strings.TrimSpace(name)

		if name == "" {
			return nil, fmt.Errorf("%w: empty rule name", ErrInvalidTag)
		}

		if !hasArg {
			arg = ""
		}

		rules = append(rules, rule{name: name, arg: arg})
	}

	return rules, nil
}

func validateNested(prefix string, fv reflect.Value) (ValidationErrors, error) {
	for fv.Kind() == reflect.Ptr {
		if fv.IsNil() {
			return nil, nil
		}
		fv = fv.Elem()
	}

	switch fv.Kind() {
	case reflect.Struct:
		errs, err := validateStruct(fv)
		if err != nil {
			return nil, err
		}

		for i := range errs {
			errs[i].Field = prefix + "." + errs[i].Field
		}

		return errs, nil

	case reflect.Slice, reflect.Array:
		var all ValidationErrors

		for i := 0; i < fv.Len(); i++ {
			errs, err := validateNested(fmt.Sprintf("%s[%d]", prefix, i), fv.Index(i))
			if err != nil {
				return nil, err
			}
			all = append(all, errs...)
		}

		return all, nil

	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.String, reflect.UnsafePointer:
		return nil, fmt.Errorf("%w: nested field %s has type %s", ErrInvalidTag, prefix, fv.Type())
	}

	return nil, fmt.Errorf("%w: nested field %s has type %s", ErrInvalidTag, prefix, fv.Type())
}

func validateValue(field string, fv reflect.Value, rules []rule) (ValidationErrors, error) {
	for fv.Kind() == reflect.Ptr {
		if fv.IsNil() {
			return nil, nil
		}
		fv = fv.Elem()
	}

	switch fv.Kind() {
	case reflect.String:
		return validateString(field, fv.String(), rules)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return validateInt(field, fv.Int(), rules)

	case reflect.Slice, reflect.Array:
		if err := validateRulesForType(field, fv.Type().Elem(), rules); err != nil {
			return nil, err
		}

		var errs ValidationErrors

		for i := 0; i < fv.Len(); i++ {
			elemErrs, err := validateValue(field, fv.Index(i), rules)
			if err != nil {
				return nil, err
			}
			errs = append(errs, elemErrs...)
		}

		return errs, nil

	case reflect.Invalid, reflect.Bool, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Struct, reflect.UnsafePointer:
		return nil, fmt.Errorf("%w: field %s has type %s", ErrUnsupportedType, field, fv.Type())
	}

	return nil, fmt.Errorf("%w: field %s has type %s", ErrUnsupportedType, field, fv.Type())
}

func validateRulesForType(field string, t reflect.Type, rules []rule) error {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		_, err := validateString(field, "", rules)
		return err

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		_, err := validateInt(field, 0, rules)
		return err

	case reflect.Slice, reflect.Array:
		return validateRulesForType(field, t.Elem(), rules)

	case reflect.Invalid, reflect.Bool, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Struct, reflect.UnsafePointer:
		return fmt.Errorf("%w: field %s has type %s", ErrUnsupportedType, field, t)
	}

	return fmt.Errorf("%w: field %s has type %s", ErrUnsupportedType, field, t)
}

func validateString(field, s string, rules []rule) (ValidationErrors, error) {
	var errs ValidationErrors

	for _, r := range rules {
		switch r.name {
		case "len":
			n, err := strconv.Atoi(strings.TrimSpace(r.arg))
			if err != nil || n < 0 {
				return nil, fmt.Errorf("%w: len:%s", ErrInvalidTag, r.arg)
			}

			actual := utf8.RuneCountInString(s)
			if actual != n {
				errs = append(errs, ValidationError{
					Field: field,
					Err:   fmt.Errorf("%w: want %d, got %d", ErrInvalidLength, n, actual),
				})
			}

		case "regexp":
			if r.arg == "" {
				return nil, fmt.Errorf("%w: empty regexp", ErrInvalidTag)
			}

			re, err := regexp.Compile(`\A(` + r.arg + `)\z`)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid regexp %q: %w", ErrInvalidTag, r.arg, err)
			}

			if !re.MatchString(s) {
				errs = append(errs, ValidationError{
					Field: field,
					Err:   fmt.Errorf("%w: %q does not match %q", ErrInvalidRegexp, s, r.arg),
				})
			}

		case "in":
			allowed := strings.Split(r.arg, ",")
			found := false

			for _, a := range allowed {
				if s == strings.TrimSpace(a) {
					found = true
					break
				}
			}

			if !found {
				errs = append(errs, ValidationError{
					Field: field,
					Err:   fmt.Errorf("%w: %q not in %v", ErrInvalidIn, s, allowed),
				})
			}

		default:
			return nil, fmt.Errorf("%w: unknown rule %q", ErrInvalidTag, r.name)
		}
	}

	return errs, nil
}

func validateInt(field string, n int64, rules []rule) (ValidationErrors, error) {
	var errs ValidationErrors

	for _, r := range rules {
		switch r.name {
		case "min":
			minimum, err := strconv.ParseInt(strings.TrimSpace(r.arg), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: min:%s", ErrInvalidTag, r.arg)
			}

			if n < minimum {
				errs = append(errs, ValidationError{
					Field: field,
					Err:   fmt.Errorf("%w: %d < %d", ErrInvalidMin, n, minimum),
				})
			}

		case "max":
			maximum, err := strconv.ParseInt(strings.TrimSpace(r.arg), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: max:%s", ErrInvalidTag, r.arg)
			}

			if n > maximum {
				errs = append(errs, ValidationError{
					Field: field,
					Err:   fmt.Errorf("%w: %d > %d", ErrInvalidMax, n, maximum),
				})
			}

		case "in":
			allowed, err := parseIntSet(r.arg)
			if err != nil {
				return nil, err
			}

			if _, ok := allowed[n]; !ok {
				errs = append(errs, ValidationError{
					Field: field,
					Err:   fmt.Errorf("%w: %d not in %s", ErrInvalidIn, n, r.arg),
				})
			}

		default:
			return nil, fmt.Errorf("%w: unknown rule %q", ErrInvalidTag, r.name)
		}
	}

	return errs, nil
}

func parseIntSet(arg string) (map[int64]struct{}, error) {
	parts := strings.Split(arg, ",")
	set := make(map[int64]struct{}, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("%w: empty number in in:%s", ErrInvalidTag, arg)
		}

		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: in:%s", ErrInvalidTag, arg)
		}

		set[n] = struct{}{}
	}

	return set, nil
}
