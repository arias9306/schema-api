package fakegen

import (
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
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
		{"credit_card", "credit_card"},
		{"card_number", "credit_card"},
		{"hex_color", "hex_color"},
		{"color", "hex_color"},
		{"ip_address", "ipv4"},
		{"ipv6", "ipv6"},
		{"ipv4", "ipv4"},
		{"mac_address", "mac_address"},
		{"mime_type", "mime_type"},
		{"file_ext", "file_extension"},
		{"extension", "file_extension"},
		{"currency_amount", "currency_amount"},
		{"price", "currency_amount"},
		{"amount", "currency_amount"},
		{"total", "currency_amount"},
		{"product_name", "product_name"},
		{"url_slug", "slug"},
		{"slug", "slug"},
		{"isbn", "isbn"},
		{"latitude", "lat"},
		{"lat", "lat"},
		{"longitude", "lng"},
		{"lng", "lng"},
		{"timezone", "timezone"},
		{"tz", "timezone"},
		{"job_title", "job_title"},
		{"company_name", "company"},
		{"company", "company"},
		{"iban", "iban"},
		{"account_number", "iban"},
		{"birth_date", "date"},
		{"dob", "date"},
		{"date", "date"},
		{"email", "email"},
		{"user_email", "email"},
		{"phone", "phone"},
		{"phone_number", "phone"},
		{"address", "address"},
		{"city", "city"},
		{"country", "country"},
		{"url", "url"},
		{"uuid", "uuid"},
		{"sentence", "sentence"},
		{"lorem", "lorem"},
		{"description", "lorem"},
		{"bio", "lorem"},
		{"summary", "lorem"},
		{"body", "lorem"},
		{"content", "lorem"},
		{"full_name", "name"},
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

func TestGenCreditCard(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		cc := GenCreditCard(rng)
		assert.Regexp(t, regexp.MustCompile(`^\d{4} \d{4} \d{4} \d{4}$`), cc)

		digits := strings.ReplaceAll(cc, " ", "")
		sum := 0
		for j := 0; j < 16; j++ {
			d, _ := strconv.Atoi(string(digits[j]))
			v := d
			if j%2 == 0 {
				v *= 2
				if v > 9 {
					v -= 9
				}
			}
			sum += v
		}
		assert.Equal(t, 0, sum%10, "credit card should pass Luhn check: %s", cc)
	}
}

func TestGenHexColor(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		assert.Regexp(t, regexp.MustCompile(`^#[0-9A-F]{6}$`), GenHexColor(rng))
	}
}

func TestGenIPv4(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		ip := GenIPv4(rng)
		parts := strings.Split(ip, ".")
		assert.Len(t, parts, 4)
		for _, p := range parts {
			n, err := strconv.Atoi(p)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, n, 0)
			assert.LessOrEqual(t, n, 255)
		}
	}
}

func TestGenIPv6(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		ip := GenIPv6(rng)
		parts := strings.Split(ip, ":")
		assert.Len(t, parts, 8)
		for _, p := range parts {
			assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{4}$`), p)
		}
	}
}

func TestGenMACAddress(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		assert.Regexp(t, regexp.MustCompile(`^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`), GenMACAddress(rng))
	}
}

func TestGenMimeType(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		m := GenMimeType(rng)
		assert.Contains(t, m, "/")
		seen[m] = true
	}
	assert.True(t, len(seen) > 1, "should generate multiple different MIME types")
}

func TestGenFileExtension(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		ext := GenFileExtension(rng)
		assert.True(t, strings.HasPrefix(ext, "."))
		assert.Greater(t, len(ext), 1)
		seen[ext] = true
	}
	assert.True(t, len(seen) > 1, "should generate multiple different extensions")
}

func TestGenCurrencyAmount(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		amt := GenCurrencyAmount(rng)
		assert.True(t, strings.HasPrefix(amt, "$"))
		parts := strings.Split(strings.TrimPrefix(amt, "$"), ".")
		assert.Len(t, parts, 2)
		assert.Equal(t, 2, len(parts[1]), "fractional part should be 2 digits: %s", amt)
	}
}

func TestGenProductName(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		p := GenProductName(rng)
		parts := strings.SplitN(p, " ", 2)
		assert.Len(t, parts, 2, "product name should have 2 parts: %s", p)
	}
}

func TestGenSlug(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		slug := GenSlug(rng)
		assert.NotEmpty(t, slug)
		assert.False(t, strings.HasPrefix(slug, "-"))
		assert.False(t, strings.HasSuffix(slug, "-"))
		assert.NotContains(t, slug, " ")
		parts := strings.Split(slug, "-")
		assert.GreaterOrEqual(t, len(parts), 3)
		assert.LessOrEqual(t, len(parts), 6)
	}
}

func TestGenWord(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		w := GenWord(rng)
		assert.NotEmpty(t, w)
		assert.Equal(t, w, strings.ToLower(w))
		seen[w] = true
	}
	assert.True(t, len(seen) > 1, "should generate multiple different words")
}

func TestGenISBN(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		isbn := GenISBN(rng)
		assert.Regexp(t, regexp.MustCompile(`^\d{3}-\d-\d{2}-\d{6}-\d$`), isbn)

		digits := strings.ReplaceAll(isbn, "-", "")
		sum := 0
		for j, ch := range digits {
			d := int(ch - '0')
			if j%2 == 0 {
				sum += d
			} else {
				sum += d * 3
			}
		}
		assert.Equal(t, 0, sum%10, "ISBN should pass check digit validation: %s", isbn)
	}
}

func TestGenLat(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		lat := GenLat(rng)
		n, err := strconv.ParseFloat(lat, 64)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, n, -90.0)
		assert.LessOrEqual(t, n, 90.0)
	}
}

func TestGenLng(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		lng := GenLng(rng)
		n, err := strconv.ParseFloat(lng, 64)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, n, -180.0)
		assert.LessOrEqual(t, n, 180.0)
	}
}

func TestGenTimezone(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		tz := GenTimezone(rng)
		assert.Contains(t, tz, "/")
		seen[tz] = true
	}
	assert.True(t, len(seen) > 1, "should generate multiple different timezones")
}

func TestGenJobTitle(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		title := GenJobTitle(rng)
		assert.NotEmpty(t, title)
		seen[title] = true
	}
	assert.True(t, len(seen) > 1, "should generate multiple different job titles")
}

func TestGenCompany(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		c := GenCompany(rng)
		parts := strings.SplitN(c, " ", 2)
		assert.Len(t, parts, 2, "company name should have 2 parts: %s", c)
	}
}

func TestGenIBAN(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		iban := GenIBAN(rng)
		assert.GreaterOrEqual(t, len(iban), 15)
		assert.Regexp(t, regexp.MustCompile(`^[A-Z]{2}\d{2}[A-Z0-9]+$`), iban)
	}
}

func TestGenDate(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		d := GenDate(rng)
		parsed, err := time.Parse("2006-01-02", d)
		require.NoError(t, err)
		assert.True(t, parsed.Before(time.Now().AddDate(0, 0, 1)))
		assert.True(t, parsed.After(time.Now().AddDate(0, 0, -731)))
	}
}

func TestGenLorem(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		text := GenLorem(rng)
		words := strings.Split(text, " ")
		assert.GreaterOrEqual(t, len(words), 6)
		assert.LessOrEqual(t, len(words), 12)
		for _, w := range words {
			assert.Equal(t, w, strings.ToLower(w))
		}
	}
}

func TestGenSentence(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		s := GenSentence(rng)
		assert.True(t, strings.HasSuffix(s, "."))
		first := string(s[0])
		assert.Equal(t, strings.ToUpper(first), first, "sentence should start with uppercase")
		words := strings.Split(strings.TrimSuffix(s, "."), " ")
		assert.GreaterOrEqual(t, len(words), 8)
		assert.LessOrEqual(t, len(words), 15)
	}
}
