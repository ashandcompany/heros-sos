package auth

import "strings"

const (
	unknownDevice   = "Appareil inconnu"
	deviceSeparator = " · "
	maxDeviceLabel  = 64
)

// browsers liste les navigateurs du plus spécifique au plus générique : Edge et Opera
// annoncent aussi « Chrome », et Chrome annonce aussi « Safari ».
var browsers = []struct{ marker, name string }{
	{"Edg/", "Edge"},
	{"EdgA/", "Edge"},
	{"OPR/", "Opera"},
	{"Firefox/", "Firefox"},
	{"FxiOS/", "Firefox"},
	{"CriOS/", "Chrome"},
	{"Chrome/", "Chrome"},
	{"Safari/", "Safari"},
}

// systems liste les systèmes du plus spécifique au plus générique : Android annonce aussi
// « Linux », iOS annonce aussi « Mac OS X ».
var systems = []struct{ marker, name string }{
	{"iPhone", "iOS"},
	{"iPad", "iOS"},
	{"Android", "Android"},
	{"CrOS", "ChromeOS"},
	{"Windows", "Windows"},
	{"Macintosh", "macOS"},
	{"Mac OS X", "macOS"},
	{"Linux", "Linux"},
}

// deviceLabel résume un User-Agent en « Navigateur · Système » pour la liste des appareils
// connectés. La chaîne brute n'est jamais conservée : elle est très précise (version, modèle)
// et peut servir à identifier une personne.
func deviceLabel(userAgent string) string {
	browser, system := "", ""
	for _, candidate := range browsers {
		if strings.Contains(userAgent, candidate.marker) {
			browser = candidate.name
			break
		}
	}
	for _, candidate := range systems {
		if strings.Contains(userAgent, candidate.marker) {
			system = candidate.name
			break
		}
	}

	var label string
	switch {
	case browser != "" && system != "":
		label = browser + deviceSeparator + system
	case browser != "":
		label = browser
	case system != "":
		label = system
	default:
		return unknownDevice
	}
	return label[:min(len(label), maxDeviceLabel)]
}
