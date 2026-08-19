package fakegen

import (
	"math"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestValueInt(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	min, max := 5.0, 10.0
	spec := Spec{Type: "int", Min: &min, Max: &max}

	for i := 0; i < 100; i++ {
		v, err := Value(rng, spec)
		require.NoError(t, err)
		n, ok := v.(int)
		require.True(t, ok)
		assert.GreaterOrEqual(t, n, 5)
		assert.LessOrEqual(t, n, 10)
	}
}

func TestValueIntDefaults(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	v, err := Value(rng, Spec{Type: "int"})
	require.NoError(t, err)
	n := v.(int)
	assert.GreaterOrEqual(t, n, 0)
	assert.LessOrEqual(t, n, 10000)
}

func TestValueIntMaxLessThanMin(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	min, max := 10.0, 5.0
	v, err := Value(rng, Spec{Type: "int", Min: &min, Max: &max})
	require.NoError(t, err)
	assert.Equal(t, 10, v)
}

func TestValueIntHugeRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	min, max := 0.0, float64(math.MaxInt32)+100.0
	v, err := Value(rng, Spec{Type: "int", Min: &min, Max: &max})
	require.NoError(t, err)
	n := v.(int)
	assert.GreaterOrEqual(t, n, 0)
	assert.Less(t, n, math.MaxInt32)
}

func TestValueFloat(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	min, max := 0.0, 100.0
	spec := Spec{Type: "float", Min: &min, Max: &max}

	for i := 0; i < 100; i++ {
		v, err := Value(rng, spec)
		require.NoError(t, err)
		f := v.(float64)
		assert.GreaterOrEqual(t, f, 0.0)
		assert.LessOrEqual(t, f, 100.0)
	}
}

func TestValueFloatMaxLessThanMin(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	min, max := 10.0, 5.0
	v, err := Value(rng, Spec{Type: "float", Min: &min, Max: &max})
	require.NoError(t, err)
	assert.Equal(t, 10.0, v)
}

func TestValueBool(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	sawTrue, sawFalse := false, false

	for i := 0; i < 200; i++ {
		v, err := Value(rng, Spec{Type: "bool"})
		require.NoError(t, err)
		b, ok := v.(bool)
		require.True(t, ok)
		if b {
			sawTrue = true
		} else {
			sawFalse = true
		}
	}

	assert.True(t, sawTrue)
	assert.True(t, sawFalse)
}

func TestValueDatetime(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	v, err := Value(rng, Spec{Type: "datetime"})
	require.NoError(t, err)
	s, ok := v.(string)
	require.True(t, ok)

	parsed, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	assert.True(t, parsed.Before(time.Now()))
	assert.True(t, parsed.After(time.Now().AddDate(0, 0, -730)))
}

func TestValueStringFormat(t *testing.T) {
	rng := rand.New(rand.NewSource(1))

	t.Run("email", func(t *testing.T) {
		v, err := Value(rng, Spec{Type: "string", Format: "email"})
		require.NoError(t, err)
		assert.Contains(t, v.(string), "@")
	})

	t.Run("uuid", func(t *testing.T) {
		v, err := Value(rng, Spec{Type: "string", Format: "uuid"})
		require.NoError(t, err)
		assert.Regexp(t, uuidV4Pattern, v.(string))
	})
}

func TestValueStringLengthBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	minLen, maxLen := 8, 12
	spec := Spec{Type: "string", MinLength: &minLen, MaxLength: &maxLen}

	for i := 0; i < 100; i++ {
		v, err := Value(rng, spec)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(v.(string)), 8)
		assert.LessOrEqual(t, len(v.(string)), 12)
	}
}

func TestValueStringSwappedLengths(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	minLen, maxLen := 10, 5
	v, err := Value(rng, Spec{Type: "string", MinLength: &minLen, MaxLength: &maxLen})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(v.(string)), 5)
	assert.LessOrEqual(t, len(v.(string)), 10)
}

func TestValueStringFormatViolatingLengthFallsBack(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	minLen, maxLen := 100, 120
	v, err := Value(rng, Spec{Type: "string", Format: "email", MinLength: &minLen, MaxLength: &maxLen})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(v.(string)), 100)
	assert.LessOrEqual(t, len(v.(string)), 120)
}

func TestValueUnsupportedType(t *testing.T) {
	_, err := Value(rand.New(rand.NewSource(1)), Spec{Type: "blob"})
	require.Error(t, err)
	assert.ErrorContains(t, err, `unsupported type "blob"`)
}

func TestResolveFormat(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"username", "username"},
		{"user_name", "username"},
		{"firstname", "firstname"},
		{"first_name", "firstname"},
		{"lastname", "lastname"},
		{"last_name", "lastname"},
		{"email", "email"},
		{"user_email", "email"},
		{"phone", "phone"},
		{"phone_number", "phone"},
		{"address", "address"},
		{"city", "city"},
		{"country", "country"},
		{"url", "url"},
		{"uuid", "uuid"},
		{"name", "name"},
		{"fullname", "name"},
		{"created_at", ""},
		{"", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolveFormat(tc.name))
		})
	}
}

func TestViolatesExplicitLength(t *testing.T) {
	short, long := 10, 5

	assert.False(t, violatesExplicitLength(Spec{}, "anything"))
	assert.True(t, violatesExplicitLength(Spec{MinLength: &short}, "short"))
	assert.False(t, violatesExplicitLength(Spec{MinLength: &short}, "longenough"))
	assert.True(t, violatesExplicitLength(Spec{MaxLength: &long}, "waytoolong"))
	assert.False(t, violatesExplicitLength(Spec{MaxLength: &long}, "ok"))
}

func TestRandomString(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	s := randomString(rng, 32)
	assert.Len(t, s, 32)
	assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9]+$`), s)
}

func TestGenEmail(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	pattern := regexp.MustCompile(`^[a-zñáéíóúü]+\.[a-zñáéíóúü]+\d+@[a-z.]+$`)
	for i := 0; i < 50; i++ {
		assert.Regexp(t, pattern, GenEmail(rng))
	}
}

func TestGenPhone(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		assert.Regexp(t, regexp.MustCompile(`^\+57 3\d+$`), GenPhone(rng))
	}
}

func TestGenUsername(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	pattern := regexp.MustCompile(`^[a-zñáéíóúü]+\d{1,2}$`)
	for i := 0; i < 50; i++ {
		assert.Regexp(t, pattern, GenUsername(rng))
	}
}

func TestGenURL(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		assert.Regexp(t, regexp.MustCompile(`^https://[a-z]+\.acme/$`), GenURL(rng))
	}
}

func TestGenAddress(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		assert.Regexp(t, regexp.MustCompile(`^(Calle|Carrera) \d+ # \d+ - \d+$`), GenAddress(rng))
	}
}

func TestGenFullName(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		name := GenFullName(rng)
		assert.Contains(t, name, " ")
		parts := regexp.MustCompile(`\s+`).Split(name, -1)
		assert.Len(t, parts, 2)
	}
}

func TestGenUUID(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		u := GenUUID(rng)
		assert.Regexp(t, uuidV4Pattern, u)
		assert.False(t, seen[u], "uuid should be unique")
		seen[u] = true
	}
}
