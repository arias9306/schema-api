// Package fakegen provides shared scalar value generation used by schema
// seeding and mock endpoint rendering.
package fakegen

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
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

var MimeTypes = []string{
	"application/json", "application/xml", "application/pdf", "application/zip",
	"text/plain", "text/html", "text/css", "text/csv",
	"image/png", "image/jpeg", "image/gif", "image/svg+xml",
	"audio/mpeg", "video/mp4",
	"multipart/form-data", "application/octet-stream",
	"application/x-www-form-urlencoded", "application/vnd.api+json",
}

var FileExtensions = []string{
	".txt", ".pdf", ".doc", ".docx", ".xls", ".xlsx",
	".png", ".jpg", ".jpeg", ".gif", ".svg",
	".mp3", ".mp4", ".avi", ".mov",
	".zip", ".tar", ".gz",
	".html", ".css", ".js", ".json", ".xml", ".csv",
}

var ProductAdjectives = []string{
	"Atomic", "Electronic", "Quantum", "Ultra", "Mega",
	"Super", "Hyper", "Turbo", "Power", "Pro",
	"Elite", "Prime", "Apex", "Summit", "Atlas",
	"Nova", "Stellar", "Cosmic", "Titan", "Vortex",
}

var ProductNouns = []string{
	"Thunder", "Falcon", "Phoenix", "Horizon", "Pioneer",
	"Vanguard", "Sentinel", "Nexus", "Zenith", "Pulse",
	"Catalyst", "Eclipse", "Vertex", "Matrix", "Cortex",
	"Prism", "Forge", "Spark", "Drift", "Surge",
}

var Timezones = []string{
	"America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles",
	"America/Bogota", "Pacific/Honolulu", "America/Toronto", "America/Vancouver",
	"Europe/London", "Europe/Paris", "Europe/Berlin", "Europe/Madrid",
	"Europe/Rome", "Europe/Amsterdam", "Europe/Moscow",
	"Asia/Tokyo", "Asia/Shanghai", "Asia/Kolkata", "Asia/Dubai", "Asia/Singapore",
	"Australia/Sydney", "Australia/Melbourne", "Pacific/Auckland",
	"Africa/Cairo", "Africa/Lagos", "America/Sao_Paulo", "America/Mexico_City",
}

var JobTitles = []string{
	"Software Engineer", "Product Manager", "Data Scientist", "DevOps Engineer",
	"UX Designer", "Frontend Developer", "Backend Developer", "Full Stack Developer",
	"Engineering Manager", "Technical Lead", "Solutions Architect", "Site Reliability Engineer",
}

var JobDepartments = []string{
	"Engineering", "Product", "Marketing", "Sales", "Finance",
	"Operations", "Design", "Research", "Support", "Legal",
	"Human Resources", "Data",
}

var CompanyAdjectives = []string{
	"Quantum", "Nexus", "Apex", "Pulse", "Stellar",
	"Cyber", "Nova", "Atlas", "Zenith", "Vortex",
	"Prime", "Vertex", "Cortex", "Titan", "Forge",
}

var CompanyNouns = []string{
	"Labs", "Tech", "Works", "Systems", "Solutions",
	"Dynamics", "Collective", "Ventures", "Digital", "Logic",
	"Innovations", "Networks", "Analytics", "Industries", "Partners",
}

var LoremWords = []string{
	"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit",
	"sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore",
	"magna", "aliqua", "enim", "ad", "minim", "veniam", "quis", "nostrud",
	"exercitation", "ullamco", "laboris", "nisi", "aliquip", "ex", "ea", "commodo",
	"consequat", "duis", "aute", "irure", "in", "reprehenderit", "voluptate",
	"velit", "esse", "cillum", "fugiat", "nulla", "pariatur", "excepteur", "sint",
	"occaecat", "cupidatat", "non", "proident", "sunt", "culpa", "qui", "officia",
	"deserunt", "mollit", "anim", "id", "est", "laborum", "perspiciatis", "unde",
	"omnis", "iste", "natus", "error", "voluptatem", "accusantium", "doloremque",
	"laudantium", "totam", "rem", "aperiam", "eaque", "ipsa", "quae", "ab", "illo",
	"inventore", "veritatis", "quasi", "architecto", "beatae", "vitae", "dicta",
}

var IBANCountries = []struct {
	Code string
	BBAN int
}{
	{"DE", 20}, {"GB", 22}, {"FR", 27}, {"ES", 24},
	{"IT", 27}, {"NL", 18}, {"BE", 16}, {"CH", 21},
	{"AT", 20}, {"SE", 24},
}

var FormatGenerators = map[string]Generator{
	"name":            GenFullName,
	"firstname":       GenFirstname,
	"lastname":        GenLastname,
	"username":        GenUsername,
	"email":           GenEmail,
	"phone":           GenPhone,
	"address":         GenAddress,
	"city":            GenCity,
	"country":         GenCountry,
	"url":             GenURL,
	"uuid":            GenUUID,
	"credit_card":     GenCreditCard,
	"hex_color":       GenHexColor,
	"ipv4":            GenIPv4,
	"ipv6":            GenIPv6,
	"mac_address":     GenMACAddress,
	"mime_type":       GenMimeType,
	"file_extension":  GenFileExtension,
	"currency_amount": GenCurrencyAmount,
	"product_name":    GenProductName,
	"slug":            GenSlug,
	"word":            GenWord,
	"isbn":            GenISBN,
	"lat":             GenLat,
	"lng":             GenLng,
	"timezone":        GenTimezone,
	"job_title":       GenJobTitle,
	"company":         GenCompany,
	"iban":            GenIBAN,
	"date":            GenDate,
	"lorem":           GenLorem,
	"sentence":        GenSentence,
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
		{"phone", "phone"},
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
	}

	for _, heuristic := range heuristics {
		if strings.Contains(name, heuristic.needle) {
			return heuristic.format
		}
	}

	return ""
}

func Value(rng *rand.Rand, spec Spec) (any, error) {
	switch spec.Type {
	case "string":
		return generateString(rng, spec), nil

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
			return int(min) + rng.Intn(math.MaxInt32), nil
		}

		return int(min) + rng.Intn(int(rangeSize)+1), nil

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

		return float64(int((min+rng.Float64()*(max-min))*100)) / 100, nil

	case "bool":
		return rng.Intn(2) == 1, nil

	case "datetime":
		days := rng.Intn(365 * 2)
		return time.Now().AddDate(0, 0, -days).Format(time.RFC3339), nil

	default:
		return nil, fmt.Errorf("unsupported type %q", spec.Type)
	}
}

func generateString(rng *rand.Rand, spec Spec) string {
	if format := strings.ToLower(spec.Format); format != "" {
		if generator, ok := FormatGenerators[format]; ok {
			value := generator(rng)
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
		length += rng.Intn(maxLength - minLength + 1)
	}

	return randomString(rng, length)
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

func GenCreditCard(rng *rand.Rand) string {
	digits := make([]int, 15)
	for i := range digits {
		digits[i] = rng.Intn(10)
	}
	sum := 0
	for i, d := range digits {
		v := d
		if (14-i)%2 == 0 {
			v *= 2
			if v > 9 {
				v -= 9
			}
		}
		sum += v
	}
	check := (10 - sum%10) % 10
	all := append(digits, check)
	return fmt.Sprintf("%d%d%d%d %d%d%d%d %d%d%d%d %d%d%d%d",
		all[0], all[1], all[2], all[3],
		all[4], all[5], all[6], all[7],
		all[8], all[9], all[10], all[11],
		all[12], all[13], all[14], all[15])
}

func GenHexColor(rng *rand.Rand) string {
	return fmt.Sprintf("#%06X", rng.Intn(0xFFFFFF+1))
}

func GenIPv4(rng *rand.Rand) string {
	return fmt.Sprintf("%d.%d.%d.%d", rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(256))
}

func GenIPv6(rng *rand.Rand) string {
	parts := make([]string, 8)
	for i := range parts {
		parts[i] = fmt.Sprintf("%04x", rng.Intn(0x10000))
	}
	return strings.Join(parts, ":")
}

func GenMACAddress(rng *rand.Rand) string {
	b := make([]byte, 6)
	rng.Read(b)
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", b[0], b[1], b[2], b[3], b[4], b[5])
}

func GenMimeType(rng *rand.Rand) string {
	return MimeTypes[rng.Intn(len(MimeTypes))]
}

func GenFileExtension(rng *rand.Rand) string {
	return FileExtensions[rng.Intn(len(FileExtensions))]
}

func GenCurrencyAmount(rng *rand.Rand) string {
	whole := rng.Intn(100000)
	frac := rng.Intn(100)
	return fmt.Sprintf("$%s.%02d", strconv.Itoa(whole), frac)
}

func GenProductName(rng *rand.Rand) string {
	return ProductAdjectives[rng.Intn(len(ProductAdjectives))] + " " + ProductNouns[rng.Intn(len(ProductNouns))]
}

func GenSlug(rng *rand.Rand) string {
	n := 3 + rng.Intn(4)
	words := make([]string, n)
	for i := range words {
		words[i] = LoremWords[rng.Intn(len(LoremWords))]
	}
	return strings.Join(words, "-")
}

func GenWord(rng *rand.Rand) string {
	return LoremWords[rng.Intn(len(LoremWords))]
}

func GenISBN(rng *rand.Rand) string {
	digits := make([]int, 12)
	for i := range digits {
		digits[i] = rng.Intn(10)
	}
	sum := 0
	for i, d := range digits {
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	check := (10 - sum%10) % 10
	return fmt.Sprintf("%d%d%d-%d-%d%d-%d%d%d%d%d%d-%d",
		digits[0], digits[1], digits[2], digits[3],
		digits[4], digits[5], digits[6], digits[7],
		digits[8], digits[9], digits[10], digits[11], check)
}

func GenLat(rng *rand.Rand) string {
	return fmt.Sprintf("%.6f", rng.Float64()*180-90)
}

func GenLng(rng *rand.Rand) string {
	return fmt.Sprintf("%.6f", rng.Float64()*360-180)
}

func GenTimezone(rng *rand.Rand) string {
	return Timezones[rng.Intn(len(Timezones))]
}

func GenJobTitle(rng *rand.Rand) string {
	return JobTitles[rng.Intn(len(JobTitles))]
}

func GenCompany(rng *rand.Rand) string {
	return CompanyAdjectives[rng.Intn(len(CompanyAdjectives))] + " " + CompanyNouns[rng.Intn(len(CompanyNouns))]
}

func GenIBAN(rng *rand.Rand) string {
	c := IBANCountries[rng.Intn(len(IBANCountries))]
	bban := make([]int, c.BBAN)
	for i := range bban {
		bban[i] = rng.Intn(10)
	}

	payload := ""
	for _, d := range bban {
		payload += strconv.Itoa(d)
	}
	payload += "0000" + c.Code

	mod := 0
	for _, ch := range payload {
		var v int
		if ch >= 'A' && ch <= 'Z' {
			v = int(ch-'A') + 10
		} else {
			v = int(ch - '0')
		}
		mod = (mod*10 + v) % 97
	}
	checkDigits := fmt.Sprintf("%02d", (98-mod)%97)

	result := c.Code + checkDigits
	for _, d := range bban {
		result += strconv.Itoa(d)
	}
	return result
}

func GenDate(rng *rand.Rand) string {
	days := rng.Intn(730)
	return time.Now().AddDate(0, 0, -days).Format("2006-01-02")
}

func GenLorem(rng *rand.Rand) string {
	n := 6 + rng.Intn(7)
	words := make([]string, n)
	for i := range words {
		words[i] = LoremWords[rng.Intn(len(LoremWords))]
	}
	return strings.Join(words, " ")
}

func GenSentence(rng *rand.Rand) string {
	n := 8 + rng.Intn(8)
	words := make([]string, n)
	for i := range words {
		words[i] = LoremWords[rng.Intn(len(LoremWords))]
	}
	w := words[0]
	words[0] = strings.ToUpper(w[:1]) + w[1:]
	return strings.Join(words, " ") + "."
}

func SpecFromMap(m map[string]any, name string) Spec {
	spec := Spec{}

	if t, ok := m["type"].(string); ok {
		spec.Type = t
	}

	if f, ok := AsFloat64(m["min"]); ok {
		spec.Min = &f
	}

	if f, ok := AsFloat64(m["max"]); ok {
		spec.Max = &f
	}

	if n, ok := AsInt(m["min_length"]); ok {
		spec.MinLength = &n
	}

	if n, ok := AsInt(m["max_length"]); ok {
		spec.MaxLength = &n
	}

	if rg, ok := m["regex"].(string); ok {
		spec.Regex = rg
	}

	if format, ok := m["format"].(string); ok {
		spec.Format = format
	}

	if def, ok := m["default"]; ok {
		spec.Default = def
	}

	if spec.Type == "string" && spec.Format == "" && name != "" {
		if format := ResolveFormat(name); format != "" {
			spec.Format = format
		}
	}

	return spec
}

func AsFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func AsInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}
