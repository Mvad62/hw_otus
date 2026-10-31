package hw09structvalidator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type UserRole string

// Test the function on different structures and other types.
type (
	User struct {
		ID     string `json:"id" validate:"len:36"`
		Name   string
		Age    int             `validate:"min:18|max:50"`
		Email  string          `validate:"regexp:^\\w+@\\w+\\.\\w+$"`
		Role   UserRole        `validate:"in:admin,stuff"`
		Phones []string        `validate:"len:11"`
		meta   json.RawMessage //nolint:unused
	}

	App struct {
		Version string `validate:"len:5"`
	}

	Token struct {
		Header    []byte
		Payload   []byte
		Signature []byte
	}

	Response struct {
		Code int    `validate:"in:200,404,500"`
		Body string `json:"omitempty"`
	}
)

// validUser возвращает User, проходящего все правила валидации.
// Длина ID — ровно 36 символов, Age в [18, 50], Email матчит regexp,
// Role из {admin, stuff}, каждый Phone — 11 символов.
func validUser() User {
	return User{
		ID:     "12345678-1234-1234-1234-123456789012",
		Age:    25,
		Email:  "user@example.com",
		Role:   "admin",
		Phones: []string{"79991234567"},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		in          interface{}
		expectedErr error // если nil — ошибок быть не должно
	}{
		{
			name: "valid user",
			in:   validUser(),
		},
		{
			name: "user ID wrong length",
			in: func() User {
				u := validUser()
				u.ID = "short"
				return u
			}(),
			expectedErr: ErrInvalidLength,
		},
		{
			name: "user age below min",
			in: func() User {
				u := validUser()
				u.Age = 17
				return u
			}(),
			expectedErr: ErrInvalidMin,
		},
		{
			name: "user age above max",
			in: func() User {
				u := validUser()
				u.Age = 51
				return u
			}(),
			expectedErr: ErrInvalidMax,
		},
		{
			name: "user invalid email",
			in: func() User {
				u := validUser()
				u.Email = "not-an-email"
				return u
			}(),
			expectedErr: ErrInvalidRegexp,
		},
		{
			name: "user invalid role",
			in: func() User {
				u := validUser()
				u.Role = "user"
				return u
			}(),
			expectedErr: ErrInvalidIn,
		},
		{
			name: "user invalid phone length",
			in: func() User {
				u := validUser()
				u.Phones = []string{"123"}
				return u
			}(),
			expectedErr: ErrInvalidLength,
		},
		{
			name: "user empty phone list",
			in: func() User {
				u := validUser()
				u.Phones = nil
				return u
			}(),
		},
		{
			name: "valid app",
			in:   App{Version: "1.0.0"},
		},
		{
			name:        "invalid app version",
			in:          App{Version: "1.0"},
			expectedErr: ErrInvalidLength,
		},
		{
			name: "token without validate tags",
			in: Token{
				Header:    []byte("h"),
				Payload:   []byte("p"),
				Signature: []byte("s"),
			},
		},
		{
			name: "valid response",
			in:   Response{Code: 200, Body: "ok"},
		},
		{
			name:        "invalid response code",
			in:          Response{Code: 418, Body: "teapot"},
			expectedErr: ErrInvalidIn,
		},
		{
			name:        "nil input",
			in:          nil,
			expectedErr: ErrNotStruct,
		},
		{
			name:        "non-struct input",
			in:          42,
			expectedErr: ErrNotStruct,
		},
		{
			name:        "string input",
			in:          "hello",
			expectedErr: ErrNotStruct,
		},
		{
			name: "pointer to valid user",
			in: func() *User {
				u := validUser()
				return &u
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			t.Parallel()

			err := Validate(tt.in)

			if tt.expectedErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("expected errors.Is(err, %v), got %v", tt.expectedErr, err)
			}
		})
	}
}

// Проверяем, что валидатор накапливает ВСЕ ошибки, а не останавливается на первой,
// и что возвращается именно ValidationErrors.
func TestValidateMultipleErrors(t *testing.T) {
	t.Parallel()

	u := User{
		ID:     "short",
		Age:    15,
		Email:  "bad",
		Role:   "guest",
		Phones: []string{"123"},
	}

	err := Validate(u)
	if err == nil {
		t.Fatal("expected errors, got nil")
	}

	var errs ValidationErrors
	if !errors.As(err, &errs) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}

	if len(errs) != 5 {
		t.Fatalf("expected 5 validation errors, got %d: %v", len(errs), errs)
	}

	want := map[string]bool{
		"ID": false, "Age": false, "Email": false, "Role": false, "Phones": false,
	}
	for _, e := range errs {
		if _, ok := want[e.Field]; !ok {
			t.Errorf("unexpected field %q in %v", e.Field, e)
			continue
		}
		want[e.Field] = true
	}
	for f, ok := range want {
		if !ok {
			t.Errorf("missing error for field %q", f)
		}
	}
}

// Проверяем, что errors.Is не даёт ложных срабатываний.
func TestValidateErrorsIs(t *testing.T) {
	t.Parallel()

	u := validUser()
	u.Age = 10

	err := Validate(u)

	if !errors.Is(err, ErrInvalidMin) {
		t.Fatalf("expected ErrInvalidMin, got %v", err)
	}
	if errors.Is(err, ErrInvalidMax) {
		t.Fatalf("did not expect ErrInvalidMax, got %v", err)
	}
	if errors.Is(err, ErrInvalidLength) {
		t.Fatalf("did not expect ErrInvalidLength, got %v", err)
	}
}

// Проверяем строковое представление ValidationErrors.
func TestValidationErrorsError(t *testing.T) {
	t.Parallel()

	errs := ValidationErrors{
		{Field: "Name", Err: fmt.Errorf("%w: bad", ErrInvalidLength)},
		{Field: "Age", Err: fmt.Errorf("%w: bad", ErrInvalidMin)},
	}

	msg := errs.Error()
	if !strings.Contains(msg, "Name") {
		t.Errorf("error message %q does not contain %q", msg, "Name")
	}
	if !strings.Contains(msg, "Age") {
		t.Errorf("error message %q does not contain %q", msg, "Age")
	}
}

// Проверяем комбинацию правил через "|" (логическое И).
func TestValidateCombination(t *testing.T) {
	t.Parallel()

	type Range struct {
		Value int `validate:"min:0|max:10"`
	}
	type StrRules struct {
		Value string `validate:"len:5|regexp:^\\d+$"`
	}

	t.Run("range valid", func(t *testing.T) {
		t.Parallel()
		if err := Validate(Range{Value: 5}); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("range below min", func(t *testing.T) {
		t.Parallel()
		err := Validate(Range{Value: -1})
		if !errors.Is(err, ErrInvalidMin) {
			t.Fatalf("expected ErrInvalidMin, got %v", err)
		}
	})

	t.Run("range above max", func(t *testing.T) {
		t.Parallel()
		err := Validate(Range{Value: 11})
		if !errors.Is(err, ErrInvalidMax) {
			t.Fatalf("expected ErrInvalidMax, got %v", err)
		}
	})

	t.Run("string both rules fail", func(t *testing.T) {
		t.Parallel()
		err := Validate(StrRules{Value: "abc"})

		var errs ValidationErrors
		if !errors.As(err, &errs) {
			t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
		}
		if len(errs) != 2 {
			t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
		}
		if !errors.Is(err, ErrInvalidLength) {
			t.Fatalf("expected ErrInvalidLength in %v", err)
		}
		if !errors.Is(err, ErrInvalidRegexp) {
			t.Fatalf("expected ErrInvalidRegexp in %v", err)
		}
	})

	t.Run("regexp must match the entire string", func(t *testing.T) {
		t.Parallel()

		type Digits struct {
			Value string `validate:"regexp:\\d+"`
		}

		if err := Validate(Digits{Value: "123"}); err != nil {
			t.Fatalf("expected digits to be valid, got %v", err)
		}

		err := Validate(Digits{Value: "abc123xyz"})
		if !errors.Is(err, ErrInvalidRegexp) {
			t.Fatalf("expected ErrInvalidRegexp, got %v", err)
		}
	})
}

func TestValidateEmptySliceRules(t *testing.T) {
	t.Parallel()

	type EmptyStrings struct {
		Values []string `validate:"regexp:["`
	}
	type EmptyInts struct {
		Values []int `validate:"unknown:rule"`
	}

	if err := Validate(EmptyStrings{}); !errors.Is(err, ErrInvalidTag) {
		t.Errorf("expected ErrInvalidTag for invalid regexp on empty slice, got %v", err)
	}

	if err := Validate(EmptyInts{}); !errors.Is(err, ErrInvalidTag) {
		t.Errorf("expected ErrInvalidTag for unknown rule on empty slice, got %v", err)
	}
}

// Дополнительное задание: поддержка nested-структур.
func TestValidateNested(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Value string `validate:"len:5"`
	}
	type Outer struct {
		Name  string `validate:"len:3"`
		Inner Inner  `validate:"nested"`
	}
	type OuterPtr struct {
		Inner *Inner `validate:"nested"`
	}
	type OuterSlice struct {
		Items []Inner `validate:"nested"`
	}

	t.Run("nested all valid", func(t *testing.T) {
		t.Parallel()
		err := Validate(Outer{Name: "abc", Inner: Inner{Value: "12345"}})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("nested inner invalid", func(t *testing.T) {
		t.Parallel()
		err := Validate(Outer{Name: "abc", Inner: Inner{Value: "1"}})

		var errs ValidationErrors
		if !errors.As(err, &errs) {
			t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
		}
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if errs[0].Field != "Inner.Value" {
			t.Errorf("expected field Inner.Value, got %q", errs[0].Field)
		}
	})

	t.Run("nested outer and inner invalid", func(t *testing.T) {
		t.Parallel()
		err := Validate(Outer{Name: "x", Inner: Inner{Value: "1"}})

		var errs ValidationErrors
		if !errors.As(err, &errs) {
			t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
		}
		if len(errs) != 2 {
			t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("nested pointer invalid", func(t *testing.T) {
		t.Parallel()
		err := Validate(OuterPtr{Inner: &Inner{Value: "1"}})

		var errs ValidationErrors
		if !errors.As(err, &errs) {
			t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
		}
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if errs[0].Field != "Inner.Value" {
			t.Errorf("expected field Inner.Value, got %q", errs[0].Field)
		}
	})

	t.Run("nested nil pointer", func(t *testing.T) {
		t.Parallel()
		err := Validate(OuterPtr{Inner: nil})
		if err != nil {
			t.Fatalf("expected no error for nil nested pointer, got %v", err)
		}
	})

	t.Run("nested slice", func(t *testing.T) {
		t.Parallel()
		err := Validate(OuterSlice{Items: []Inner{{Value: "1"}, {Value: "12345"}}})

		var errs ValidationErrors
		if !errors.As(err, &errs) {
			t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
		}
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if !strings.HasPrefix(errs[0].Field, "Items[0]") {
			t.Errorf("expected field starting with Items[0], got %q", errs[0].Field)
		}
	})
}
