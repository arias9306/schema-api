package seed

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/arias9306/schema-api/schema"
)

var firstNames = []string{
	"Andres", "Felipe", "Moisés", "Sergio", "Daniel", "Pedro", "Darwin",
	"Marcos", "Julian", "Javier", "Pancracio", "Patroclo", "Daniela",
	"Maria", "Mariela", "Laura", "David", "Juan", "Karen", "Diana",
	"Jennifer", "Alvaro", "Alberto", "Sara", "Patricia", "Mateo",
	"Paula", "Carlos", "Miguel", "Lucia", "Sofia", "Claudia", "Claudio",
	"Chipiti", "Rocio", "Amparo", "Ana", "Venecia", "Ivan",
}

var lastNames = []string{
	"Arias", "Diaz", "García", "González", "Andrade", "Fernández", "López",
	"Martínez", "Moreno", "Morris", "Muñoz", "Romero", "Baines", "Rojas",
	"Alvarez", "Sánchez", "Pérez", "Dorante", "Rios", "Moncada", "Restrepo",
	"Pinto", "Triana", "Rosales", "Vasquez", "Almeida", "Hernández", "Solano",
	"Jaimes", "Rodriguez", "Pardo", "Mantilla", "Estupiñan", "Niño", "Narvaez",
	"Polo", "Padilla", "Pinzón", "Pradilla", "Porras", "Arenas", "Correa", "Carrillo",
	"Silva", "Ruiz", "Torres", "Vega", "Pereira", "Medina", "Luna",
}

var emailDomains = []string{
	"gmail.com", "yahoo.com", "outlook.com", "hotmail.com", "protonmail.com",
	"icloud.com", "company.com", "example.org", "mail.com", "aol.com", "acme.com",
}

var cities = []string{
	"Bucaramanga", "Bogotá", "Medellín", "Cali", "Barranquilla", "Cartagena",
	"Santa Marta", "Pereira", "Manizales", "Cúcuta",
	"Villavicencio", "Ibagué", "Pasto", "Montería", "Neiva",
	"Armenia", "Sincelejo", "Popayán", "Tunja", "Valledupar",
}

var countries = []string{
	"Colombia", "Alemania", "Argentina", "Brasil", "Chile", "México",
	"Perú", "Ecuador", "Venezuela", "Uruguay", "Paraguay",
	"Bolivia", "Panamá", "Costa Rica", "Honduras", "Guatemala",
	"El Salvador", "Nicaragua", "República Dominicana", "Cuba", "España",
}

var streetNames = []string{"Calle", "Carrera"}

type generator func(randomizer *rand.Rand) string

var formatGenerators = map[string]generator{
	"name":      genFullName,
	"firstname": genFirstname,
	"lastname":  genLastname,
	"username":  genUsername,
	"email":     genEmail,
	"phone":     genPhone,
	"address":   genAddress,
	"city":      genCity,
	"country":   genCountry,
	"url":       genURL,
	"uuid":      genUUID,
}

func resolveFormat(column schema.Column) string {

	if column.Format != "" {
		return strings.ToLower(column.Format)
	}

	if column.Type != "string" {
		return ""
	}

	name := strings.ToLower(column.Name)

	heuristics := []struct {
		needle string
		format string
	}{
		{"name", "name"},
		{"email", "email"},
		{"user_name", "username"},
		{"username", "username"},
		{"first_name", "firstname"},
		{"firstname", "firstname"},
		{"last_name", "lastname"},
		{"lastname", "lastname"},
		{"phone", "phone"},
		{"address", "address"},
		{"city", "city"},
		{"country", "country"},
		{"url", "url"},
		{"uuid", "uuid"},
	}

	for _, heuristic := range heuristics {
		if strings.Contains(name, heuristic.needle) {
			return heuristic.format
		}
	}

	return ""
}

func genFirstname(randomizer *rand.Rand) string {
	return firstNames[randomizer.Intn(len(firstNames))]
}

func genLastname(randomizer *rand.Rand) string {
	return lastNames[randomizer.Intn(len(lastNames))]
}

func genUsername(randomizer *rand.Rand) string {
	return strings.ToLower(genFirstname(randomizer)+genFirstname(randomizer)) + fmt.Sprintf("%d", randomizer.Intn(99))
}

func genEmail(randomizer *rand.Rand) string {
	first := strings.ToLower(genFirstname(randomizer))
	last := strings.ToLower(genLastname(randomizer))
	domain := emailDomains[randomizer.Intn(len(emailDomains))]
	return fmt.Sprintf("%s.%s%d@%s", first, last, randomizer.Intn(1000), domain)
}

func genFullName(randomizer *rand.Rand) string {
	return genFirstname(randomizer) + " " + genLastname(randomizer)
}

func genPhone(randomizer *rand.Rand) string {
	return fmt.Sprintf("+57 3%04d", randomizer.Intn(999999999))
}

func genAddress(randomizer *rand.Rand) string {
	return fmt.Sprintf("%s %d # %d - %d", streetNames[randomizer.Intn(len(streetNames))], randomizer.Intn(100), randomizer.Intn(100), randomizer.Intn(100))
}

func genCity(randomizer *rand.Rand) string {
	return cities[randomizer.Intn(len(cities))]
}

func genCountry(randomizer *rand.Rand) string {
	return countries[randomizer.Intn(len(countries))]
}

func genURL(randomizer *rand.Rand) string {
	word := strings.Split(emailDomains[randomizer.Intn(len(emailDomains))], ".")[0]
	return fmt.Sprintf("https://%s.acme/", word)
}

func genUUID(randomizer *rand.Rand) string {
	b := make([]byte, 16)
	randomizer.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
