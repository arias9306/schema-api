// Package fakegen provides shared scalar value generation used by schema
// seeding and mock endpoint rendering.
package fakegen

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

type Spec struct {
	Type      string
	Min       *float64
	Max       *float64
	MinLength *int
	MaxLength *int
	Regex     string
	Format    string
	Default   any
}

type Generator func(rng *rand.Rand) string

var FirstNames = []string{
	"Andres", "Felipe", "Moisés", "Sergio", "Daniel", "Pedro", "Darwin",
	"Marcos", "Julian", "Javier", "Pancracio", "Patroclo", "Daniela",
	"Maria", "Mariela", "Laura", "David", "Juan", "Karen", "Diana",
	"Jennifer", "Alvaro", "Alberto", "Sara", "Patricia", "Mateo",
	"Paula", "Carlos", "Miguel", "Lucia", "Sofia", "Claudia", "Claudio",
	"Chipiti", "Rocio", "Amparo", "Ana", "Venecia", "Ivan",
}

var LastNames = []string{
	"Arias", "Diaz", "García", "González", "Andrade", "Fernández", "López",
	"Martínez", "Moreno", "Morris", "Muñoz", "Romero", "Baines", "Rojas",
	"Alvarez", "Sánchez", "Pérez", "Dorante", "Rios", "Moncada", "Restrepo",
	"Pinto", "Triana", "Rosales", "Vasquez", "Almeida", "Hernández", "Solano",
	"Jaimes", "Rodriguez", "Pardo", "Mantilla", "Estupiñan", "Niño", "Narvaez",
	"Polo", "Padilla", "Pinzón", "Pradilla", "Porras", "Arenas", "Correa", "Carrillo",
	"Silva", "Ruiz", "Torres", "Vega", "Pereira", "Medina", "Luna",
}

var EmailDomains = []string{
	"gmail.com", "yahoo.com", "outlook.com", "hotmail.com", "protonmail.com",
	"icloud.com", "company.com", "example.org", "mail.com", "aol.com", "acme.com",
}

var Cities = []string{
	"Bucaramanga", "Bogotá", "Medellín", "Cali", "Barranquilla", "Cartagena",
	"Santa Marta", "Pereira", "Manizales", "Cúcuta",
	"Villavicencio", "Ibagué", "Pasto", "Montería", "Neiva",
	"Armenia", "Sincelejo", "Popayán", "Tunja", "Valledupar",
}

var Countries = []string{
	"Colombia", "Alemania", "Argentina", "Brasil", "Chile", "México",
	"Perú", "Ecuador", "Venezuela", "Uruguay", "Paraguay",
	"Bolivia", "Panamá", "Costa Rica", "Honduras", "Guatemala",
	"El Salvador", "Nicaragua", "República Dominicana", "Cuba", "España",
}

var StreetNames = []string{"Calle", "Carrera"}

var FormatGenerators = map[string]Generator{
	"name":      GenFullName,
	"firstname": GenFirstname,
	"lastname":  GenLastname,
	"username":  GenUsername,
	"email":     GenEmail,
	"phone":     GenPhone,
	"address":   GenAddress,
	"city":      GenCity,
	"country":   GenCountry,
	"url":       GenURL,
	"uuid":      GenUUID,
}

func ResolveFormat(name string) string {
	name = strings.ToLower(name)

	heuristics := []struct {
		needle string
		format string
	}{
		{"user_name", "username"},
		{"username", "username"},
		{"first_name", "firstname"},
		{"firstname", "firstname"},
		{"last_name", "lastname"},
		{"lastname", "lastname"},
		{"email", "email"},
		{"phone", "phone"},
		{"address", "address"},
		{"city", "city"},
		{"country", "country"},
		{"url", "url"},
		{"uuid", "uuid"},
		{"name", "name"},
	}

	for _, heuristic := range heuristics {
		if strings.Contains(name, heuristic.needle) {
			return heuristic.format
		}
	}

	return ""
}

func Value(randomizer *rand.Rand, spec Spec) (any, error) {
	switch spec.Type {
	case "string":
		return generateString(randomizer, spec), nil

	case "int":
		min := 0.0
		max := 10000.0

		if spec.Min != nil {
			min = *spec.Min
		}

		if spec.Max != nil {
			max = *spec.Max
		}

		if max <= min {
			return int(min), nil
		}

		rangeSize := max - min
		if rangeSize >= math.MaxInt32 {
			return int(min) + randomizer.Intn(math.MaxInt32), nil
		}

		return int(min) + randomizer.Intn(int(rangeSize)+1), nil

	case "float":
		min := 0.0
		max := 10000.0

		if spec.Min != nil {
			min = *spec.Min
		}

		if spec.Max != nil {
			max = *spec.Max
		}

		if max <= min {
			return min, nil
		}

		return float64(int((min+randomizer.Float64()*(max-min))*100)) / 100, nil

	case "bool":
		return randomizer.Intn(2) == 1, nil

	case "datetime":
		days := randomizer.Intn(365 * 2)
		return time.Now().AddDate(0, 0, -days).Format(time.RFC3339), nil

	default:
		return nil, fmt.Errorf("unsupported type %q", spec.Type)
	}
}

func generateString(randomizer *rand.Rand, spec Spec) string {
	if format := strings.ToLower(spec.Format); format != "" {
		if generator, ok := FormatGenerators[format]; ok {
			value := generator(randomizer)
			if !violatesExplicitLength(spec, value) {
				return value
			}
		}
	}

	minLength := 5
	maxLength := 30

	if spec.MinLength != nil {
		minLength = *spec.MinLength
	}

	if spec.MaxLength != nil {
		maxLength = *spec.MaxLength
	}

	if minLength > maxLength {
		minLength, maxLength = maxLength, minLength
	}

	length := minLength
	if maxLength > minLength {
		length += randomizer.Intn(maxLength - minLength + 1)
	}

	return randomString(randomizer, length)
}

func violatesExplicitLength(spec Spec, value string) bool {
	if spec.MinLength != nil && len(value) < *spec.MinLength {
		return true
	}

	if spec.MaxLength != nil && len(value) > *spec.MaxLength {
		return true
	}

	return false
}

func randomString(rng *rand.Rand, n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

func GenFirstname(rng *rand.Rand) string {
	return FirstNames[rng.Intn(len(FirstNames))]
}

func GenLastname(rng *rand.Rand) string {
	return LastNames[rng.Intn(len(LastNames))]
}

func GenUsername(rng *rand.Rand) string {
	return strings.ToLower(GenFirstname(rng)+GenFirstname(rng)) + fmt.Sprintf("%d", rng.Intn(99))
}

func GenEmail(rng *rand.Rand) string {
	first := strings.ToLower(GenFirstname(rng))
	last := strings.ToLower(GenLastname(rng))
	domain := EmailDomains[rng.Intn(len(EmailDomains))]
	return fmt.Sprintf("%s.%s%d@%s", first, last, rng.Intn(1000), domain)
}

func GenFullName(rng *rand.Rand) string {
	return GenFirstname(rng) + " " + GenLastname(rng)
}

func GenPhone(rng *rand.Rand) string {
	return fmt.Sprintf("+57 3%04d", rng.Intn(999999999))
}

func GenAddress(rng *rand.Rand) string {
	return fmt.Sprintf("%s %d # %d - %d", StreetNames[rng.Intn(len(StreetNames))], rng.Intn(100), rng.Intn(100), rng.Intn(100))
}

func GenCity(rng *rand.Rand) string {
	return Cities[rng.Intn(len(Cities))]
}

func GenCountry(rng *rand.Rand) string {
	return Countries[rng.Intn(len(Countries))]
}

func GenURL(rng *rand.Rand) string {
	word := strings.Split(EmailDomains[rng.Intn(len(EmailDomains))], ".")[0]
	return fmt.Sprintf("https://%s.acme/", word)
}

func GenUUID(rng *rand.Rand) string {
	b := make([]byte, 16)
	rng.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
