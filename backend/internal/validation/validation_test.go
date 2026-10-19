package validation

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)

func TestEmail(t *testing.T) {
	tests := []struct {
		input          string
		wantNormalized string
		wantCode       string
	}{
		{"Steve.Rogers@Avengers.example", "steve.rogers@avengers.example", ""},
		{"  tony@stark.io  ", "tony@stark.io", ""},
		{"", "", CodeRequired},
		{"   ", "", CodeRequired},
		{"sans-arobase", "sans-arobase", CodeInvalid},
		{"a@b", "a@b", CodeInvalid},
		{"a @b.co", "a @b.co", CodeInvalid},
		{"Nom <a@b.co>", "nom <a@b.co>", CodeInvalid},
		{"a@@b.co", "a@@b.co", CodeInvalid},
		{strings.Repeat("a", MaxEmailLength) + "@b.co", strings.Repeat("a", MaxEmailLength) + "@b.co", CodeTooLong},
	}
	for _, test := range tests {
		normalized, code := Email(test.input)
		if normalized != test.wantNormalized || code != test.wantCode {
			t.Errorf("Email(%q) = (%q, %q), attendu (%q, %q)", test.input, normalized, code, test.wantNormalized, test.wantCode)
		}
	}
}

func TestName(t *testing.T) {
	tests := map[string]string{
		"Steve":                 "",
		"Anne-Marie":            "",
		"d'Artagnan":            "",
		"Éloïse":                "",
		"":                      CodeRequired,
		"  ":                    CodeRequired,
		"4chan":                 CodeInvalid,
		"-Marie":                CodeInvalid,
		"Bob<script>":           CodeInvalid,
		strings.Repeat("a", 51): CodeInvalid,
	}
	for input, want := range tests {
		if _, code := Name(input); code != want {
			t.Errorf("Name(%q) = %q, attendu %q", input, code, want)
		}
	}
}

func TestCivility(t *testing.T) {
	for _, valid := range []string{CivilityMadame, CivilityMonsieur, CivilityUnspecific} {
		if code := Civility(valid); code != "" {
			t.Errorf("Civility(%q) = %q", valid, code)
		}
	}
	if Civility("") != CodeRequired || Civility("monsieur-madame") != CodeInvalid {
		t.Error("les civilités vides ou inconnues doivent être refusées")
	}
}

func TestBirthDate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1990-05-17", ""},
		{"2026-10-19", ""}, // aujourd'hui
		{"1906-10-19", ""}, // exactement 120 ans
		{"", CodeRequired},
		{"2026-10-20", CodeInvalid}, // demain
		{"1906-10-18", CodeInvalid}, // plus de 120 ans
		{"17/05/1990", CodeInvalid},
		{"2026-02-30", CodeInvalid},
		{"n'importe quoi", CodeInvalid},
	}
	for _, test := range tests {
		date, code := BirthDate(test.input, fixedNow)
		if code != test.want {
			t.Errorf("BirthDate(%q) = %q, attendu %q", test.input, code, test.want)
		}
		if code == "" && FormatBirthDate(date) != test.input {
			t.Errorf("aller-retour de %q : %q", test.input, FormatBirthDate(date))
		}
	}
}

func TestPhone(t *testing.T) {
	tests := []struct {
		input          string
		wantNormalized string
		wantCode       string
	}{
		{"06 12 34 56 78", "0612345678", ""},
		{"07.12.34.56.78", "0712345678", ""},
		{"+33 6 12 34 56 78", "+33612345678", ""},
		{"+1-202-555-0143", "+12025550143", ""},
		{"", "", CodeRequired},
		{"0112345678", "0112345678", CodeInvalid}, // fixe
		{"06123", "06123", CodeInvalid},
		{"+123", "+123", CodeInvalid},
		{"abcdefghij", "abcdefghij", CodeInvalid},
	}
	for _, test := range tests {
		normalized, code := Phone(test.input)
		if normalized != test.wantNormalized || code != test.wantCode {
			t.Errorf("Phone(%q) = (%q, %q), attendu (%q, %q)", test.input, normalized, code, test.wantNormalized, test.wantCode)
		}
	}
}

func TestPostalCodeDependsOnCountry(t *testing.T) {
	tests := []struct {
		code, country string
		want          string
	}{
		{"75001", "FR", ""},
		{"7500", "FR", CodeInvalid},
		{"SW1A 1AA", "FR", CodeInvalid},
		{"SW1A 1AA", "GB", ""},
		{"1000", "BE", ""},
		{"", "FR", CodeRequired},
		{"!!", "GB", CodeInvalid},
	}
	for _, test := range tests {
		if _, got := PostalCode(test.code, test.country); got != test.want {
			t.Errorf("PostalCode(%q, %q) = %q, attendu %q", test.code, test.country, got, test.want)
		}
	}
}

func TestCountry(t *testing.T) {
	if got, code := Country(" fr "); got != "FR" || code != "" {
		t.Errorf("Country(\" fr \") = (%q, %q)", got, code)
	}
	for _, invalid := range []string{"FRA", "F", "12", "F1"} {
		if _, code := Country(invalid); code != CodeInvalid {
			t.Errorf("Country(%q) = %q, attendu %q", invalid, code, CodeInvalid)
		}
	}
	if _, code := Country(""); code != CodeRequired {
		t.Errorf("Country(\"\") = %q", code)
	}
}

func TestPassword(t *testing.T) {
	policy := PasswordPolicy{MinLength: 10, MaxLength: 20}
	tests := []struct {
		input string
		want  string
	}{
		{"Abcdefgh12", ""},
		{"Éléphant2025!", ""},
		{"", CodeRequired},
		{"Abc12", CodeTooShort},
		{"abcdefgh12", CodeWeak}, // sans majuscule
		{"ABCDEFGH12", CodeWeak}, // sans minuscule
		{"Abcdefghij", CodeWeak}, // sans chiffre
		{strings.Repeat("Ab1", 8), CodeTooLong},
		{"  Abcdefg1 ", ""}, // les espaces font partie du mot de passe
	}
	for _, test := range tests {
		if got := Password(test.input, policy); got != test.want {
			t.Errorf("Password(%q) = %q, attendu %q", test.input, got, test.want)
		}
	}
}

func TestAlias(t *testing.T) {
	tests := map[string]string{
		"Iron Man":              "",
		"Spider-Man_2":          "",
		"Hawkeye":               "",
		"":                      CodeRequired,
		"X":                     CodeInvalid,
		"admin@evil":            CodeInvalid,
		strings.Repeat("a", 31): CodeInvalid,
	}
	for input, want := range tests {
		if _, code := Alias(input); code != want {
			t.Errorf("Alias(%q) = %q, attendu %q", input, code, want)
		}
	}
}

func TestIdentifierDistinguishesEmailFromAlias(t *testing.T) {
	if normalized, code := Identifier("  Tony@Stark.IO "); normalized != "tony@stark.io" || code != "" {
		t.Errorf("email : (%q, %q)", normalized, code)
	}
	if normalized, code := Identifier("Iron Man"); normalized != "Iron Man" || code != "" {
		t.Errorf("pseudo : (%q, %q)", normalized, code)
	}
	if _, code := Identifier("pas@email"); code != CodeInvalid {
		t.Errorf("email invalide : %q", code)
	}
	if _, code := Identifier(""); code != CodeRequired {
		t.Errorf("vide : %q", code)
	}
}

func TestRequiredAndOptionalText(t *testing.T) {
	if _, code := RequiredText("  ", 10); code != CodeRequired {
		t.Errorf("RequiredText vide = %q", code)
	}
	if _, code := RequiredText(strings.Repeat("é", 11), 10); code != CodeTooLong {
		t.Errorf("RequiredText trop long = %q (les runes doivent être comptées, pas les octets)", code)
	}
	if got, code := OptionalText("  ", 10); got != "" || code != "" {
		t.Errorf("OptionalText vide = (%q, %q)", got, code)
	}
	if _, code := OptionalText(strings.Repeat("a", 11), 10); code != CodeTooLong {
		t.Errorf("OptionalText trop long = %q", code)
	}
}

func TestCollector(t *testing.T) {
	var collector Collector
	if collector.Err() != nil {
		t.Fatal("un collecteur vide ne doit pas produire d'erreur")
	}

	collector.Add("email", "")
	collector.Add("email", CodeInvalid)
	collector.Add("email", CodeRequired) // la première erreur est conservée
	collector.Add("password", CodeWeak)

	var validationErr *Error
	if err := collector.Err(); !errors.As(err, &validationErr) {
		t.Fatalf("Err() = %v, attendu *Error", err)
	}
	if validationErr.Fields["email"] != CodeInvalid || validationErr.Fields["password"] != CodeWeak {
		t.Errorf("champs = %v", validationErr.Fields)
	}
	if got, want := validationErr.Error(), "validation : email=invalid, password=weak"; got != want {
		t.Errorf("Error() = %q, attendu %q", got, want)
	}
}
