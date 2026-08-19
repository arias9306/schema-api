package validation

import (
	"testing"
	"time"

	"github.com/arias9306/schema-api/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTable() schema.Table {
	return schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{Name: "name", Type: "string", Required: true, MinLength: ptr(2), MaxLength: ptr(5)},
			{Name: "email", Type: "string", Regex: "^[^@]+@[^@]+$"},
			{Name: "age", Type: "int", Min: ptr(18.0), Max: ptr(99.0)},
			{Name: "score", Type: "float", Min: ptr(0.0), Max: ptr(10.0)},
			{Name: "active", Type: "bool"},
			{Name: "joined", Type: "datetime"},
			{Name: "nickname", Type: "string", Default: "anon"},
			{Name: "created_at", Type: "datetime", Default: "now"},
		},
	}
}

func validData() map[string]any {
	return map[string]any{
		"name":   "Alice",
		"email":  "alice@example.com",
		"age":    float64(30),
		"score":  float64(9.5),
		"active": true,
		"joined": "2026-01-02T03:04:05Z",
	}
}

func TestValidateCreateValid(t *testing.T) {
	errors, cleaned := ValidateCreate(testTable(), validData())
	require.False(t, errors.HasErrors())

	for key, value := range validData() {
		assert.Equal(t, value, cleaned[key], "field %s", key)
	}

	assert.Equal(t, "anon", cleaned["nickname"])

	createdAt, ok := cleaned["created_at"].(string)
	require.True(t, ok, "created_at default should be applied")
	_, err := time.Parse(time.RFC3339, createdAt)
	require.NoError(t, err)
}

func TestValidateCreateRequired(t *testing.T) {
	data := validData()
	delete(data, "name")

	errors, _ := ValidateCreate(testTable(), data)
	require.True(t, errors.HasErrors())
	assert.Contains(t, errors.Errors, "name is required")
}

func TestValidateCreateString(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{name: "wrong type", value: 42, want: "name: must be a string"},
		{name: "too short", value: "A", want: "name: length must be at least 2"},
		{name: "too long", value: "Toolong", want: "name: length must be at most 5"},
		{name: "regex mismatch", value: "not-an-email", want: "email: does not match pattern"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := validData()
			if tc.name == "regex mismatch" {
				data["email"] = tc.value
			} else {
				data["name"] = tc.value
			}

			errors, _ := ValidateCreate(testTable(), data)
			require.True(t, errors.HasErrors())
			assert.Contains(t, errors.Errors, tc.want)
		})
	}
}

func TestValidateCreateInt(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{name: "wrong type", value: "30", want: "age: must be a number"},
		{name: "not integer", value: float64(30.5), want: "age: must be an integer"},
		{name: "below min", value: float64(17), want: "age: must be at least 18"},
		{name: "above max", value: float64(100), want: "age: must be at most 99"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := validData()
			data["age"] = tc.value

			errors, _ := ValidateCreate(testTable(), data)
			require.True(t, errors.HasErrors())
			assert.Contains(t, errors.Errors, tc.want)
		})
	}
}

func TestValidateCreateFloat(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{name: "wrong type", value: "9.5", want: "score: must be a number"},
		{name: "below min", value: float64(-0.1), want: "score: must be at least 0"},
		{name: "above max", value: float64(10.5), want: "score: must be at most 10"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := validData()
			data["score"] = tc.value

			errors, _ := ValidateCreate(testTable(), data)
			require.True(t, errors.HasErrors())
			assert.Contains(t, errors.Errors, tc.want)
		})
	}
}

func TestValidateCreateBool(t *testing.T) {
	data := validData()
	data["active"] = "yes"

	errors, _ := ValidateCreate(testTable(), data)
	require.True(t, errors.HasErrors())
	assert.Contains(t, errors.Errors, "active: must be a boolean")
}

func TestValidateCreateDatetime(t *testing.T) {
	t.Run("valid RFC3339", func(t *testing.T) {
		data := validData()
		data["joined"] = "2026-01-02T03:04:05Z"

		errors, _ := ValidateCreate(testTable(), data)
		assert.False(t, errors.HasErrors())
	})

	t.Run("valid custom format", func(t *testing.T) {
		data := validData()
		data["joined"] = "1993-05-06 10:03:06"

		errors, _ := ValidateCreate(testTable(), data)
		assert.False(t, errors.HasErrors())
	})

	t.Run("valid date only", func(t *testing.T) {
		data := validData()
		data["joined"] = "2019-06-12"

		errors, _ := ValidateCreate(testTable(), data)
		assert.False(t, errors.HasErrors())
	})

	t.Run("invalid", func(t *testing.T) {
		data := validData()
		data["joined"] = "not-a-date"

		errors, _ := ValidateCreate(testTable(), data)
		require.True(t, errors.HasErrors())
		assert.Contains(t, errors.Errors, "joined: must be a valid datetime (RFC3339 or YYYY-MM-DD HH:MM:SS)")
	})

	t.Run("wrong type", func(t *testing.T) {
		data := validData()
		data["joined"] = float64(123)

		errors, _ := ValidateCreate(testTable(), data)
		require.True(t, errors.HasErrors())
		assert.Contains(t, errors.Errors, "joined: must be a string")
	})
}

func TestValidateCreateDefaults(t *testing.T) {
	data := validData()
	delete(data, "nickname")
	delete(data, "created_at")

	errors, cleaned := ValidateCreate(testTable(), data)
	require.False(t, errors.HasErrors())

	assert.Equal(t, "anon", cleaned["nickname"])

	createdAt, ok := cleaned["created_at"].(string)
	require.True(t, ok, "created_at default should be applied")
	parsed, err := time.Parse(time.RFC3339, createdAt)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), parsed, 2*time.Second)
}

func TestValidateCreateUnknownFieldsDoNotError(t *testing.T) {
	data := validData()
	data["unknown_column"] = "ignored"

	errors, _ := ValidateCreate(testTable(), data)
	assert.False(t, errors.HasErrors())
}

func TestValidateUpdateValidPartial(t *testing.T) {
	data := map[string]any{"age": float64(31)}
	errors := ValidateUpdate(testTable(), data)
	assert.False(t, errors.HasErrors())
}

func TestValidateUpdateInvalidValue(t *testing.T) {
	data := map[string]any{"age": float64(10)}
	errors := ValidateUpdate(testTable(), data)
	require.True(t, errors.HasErrors())
	assert.Contains(t, errors.Errors, "age: must be at least 18")
}

func TestValidateUpdateMissingRequiredAllowed(t *testing.T) {
	data := map[string]any{"email": "new@example.com"}
	errors := ValidateUpdate(testTable(), data)
	assert.False(t, errors.HasErrors())
}

func TestValidationError(t *testing.T) {
	v := &ValidationError{}
	assert.False(t, v.HasErrors())

	v.Add("something %s wrong", "went")
	require.True(t, v.HasErrors())
	assert.Equal(t, []string{"something went wrong"}, v.Errors)
	assert.Equal(t, "validation failed: something went wrong", v.Error())
}

func ptr[T any](v T) *T {
	return &v
}
