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

type generator func(randomizer *rand.Rand) string

var formatGenerators = map[string]generator{
	"username":  genUsername,
	"firstname": genFirstname,
	"lastname":  genLastname,
	"email":     genEmail,
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
		{"email", "email"},
		{"user_name", "username"},
		{"username", "username"},
		{"first_name", "firstname"},
		{"firstname", "firstname"},
		{"last_name", "lastname"},
		{"lastname", "lastname"},
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
