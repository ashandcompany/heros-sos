package server_test

import (
	"net/http"
	"sort"
	"testing"
)

// requestExample retourne l'exemple de corps JSON que le contrat propose pour l'opération : celui que
// Swagger UI pré-remplit dans « Try it out ». S'il y en a plusieurs, le premier par ordre alphabétique.
func (c *contract) requestExample(t *testing.T, operationID string) map[string]any {
	t.Helper()
	for _, item := range c.doc.Paths.Map() {
		for _, operation := range item.Operations() {
			if operation.OperationID != operationID {
				continue
			}
			if operation.RequestBody == nil || operation.RequestBody.Value == nil {
				t.Fatalf("%s n'a pas de corps de requête", operationID)
			}
			media := operation.RequestBody.Value.Content.Get("application/json")
			if media == nil || len(media.Examples) == 0 {
				t.Fatalf("%s n'a aucun exemple de requête dans openapi.yaml", operationID)
			}
			names := make([]string, 0, len(media.Examples))
			for name := range media.Examples {
				names = append(names, name)
			}
			sort.Strings(names)
			value, ok := media.Examples[names[0]].Value.Value.(map[string]any)
			if !ok {
				t.Fatalf("l'exemple %q de %s n'est pas un objet JSON", names[0], operationID)
			}
			return value
		}
	}
	t.Fatalf("opération %s introuvable dans openapi.yaml", operationID)
	return nil
}

// TestRequestExamplesOfTheContractAreAcceptedByTheAPI envoie à la vraie application chaque exemple de
// requête affiché par Swagger UI. Un exemple que l'API refuserait induirait en erreur celui qui l'essaie :
// ils doivent donc rester exacts, comme le reste du contrat.
func TestRequestExamplesOfTheContractAreAcceptedByTheAPI(t *testing.T) {
	e := newE2E(t, nil)
	example := func(operationID string) map[string]any { return e.spec.requestExample(t, operationID) }

	// Cycle d'un citoyen : inscription, activation (avec le vrai jeton du mail), connexion.
	citizen := e.newActor()
	register := example("register")
	if got := citizen.post("/auth/register", register); got.status != http.StatusCreated {
		t.Fatalf("inscription avec l'exemple : %d %s", got.status, got.body)
	}
	if got := citizen.post("/auth/activate", map[string]string{"token": e.lastToken()}); got.status != http.StatusNoContent {
		t.Fatalf("activation : %d %s", got.status, got.body)
	}

	login := example("login")
	if login["identifier"] != register["email"] || login["password"] != register["password"] {
		t.Errorf("l'exemple de connexion doit être celui du compte de l'exemple d'inscription : %v / %v", login, register)
	}
	loggedIn := citizen.post("/auth/login", login)
	if loggedIn.status != http.StatusOK {
		t.Fatalf("connexion avec l'exemple : %d %s", loggedIn.status, loggedIn.body)
	}
	var session struct{ CSRFToken string }
	loggedIn.decode(t, &session)
	citizen.csrf = session.CSRFToken

	// Demande de statut, puis refus par un modérateur.
	if got := citizen.post("/heroes/requests", example("requestHeroStatus")); got.status != http.StatusCreated {
		t.Fatalf("demande de statut avec l'exemple : %d %s", got.status, got.body)
	}
	moderator := e.newModerator("modo@example.org")
	reject := moderator.post("/admin/hero-requests/"+citizen.userID()+"/reject", example("rejectHeroRequest"))
	if reject.status != http.StatusNoContent {
		t.Errorf("refus avec l'exemple : %d %s", reject.status, reject.body)
	}

	// Routes publiques : la réponse est la même pour tout le monde.
	anonymous := e.newActor()
	for operationID, path := range map[string]string{
		"resendActivation":     "/auth/resend-activation",
		"requestPasswordReset": "/auth/password-reset",
	} {
		if got := anonymous.post(path, example(operationID)); got.status != http.StatusAccepted {
			t.Errorf("%s avec l'exemple : %d %s", operationID, got.status, got.body)
		}
	}

	// Les jetons d'exemple sont un texte à remplacer : ils doivent être refusés comme jetons invalides,
	// et non comme un corps mal formé ou des données invalides (le reste de l'exemple est correct).
	for operationID, path := range map[string]string{
		"activate":             "/auth/activate",
		"confirmPasswordReset": "/auth/password-reset/confirm",
	} {
		got := anonymous.post(path, example(operationID))
		if got.status != http.StatusBadRequest || got.code() != "invalid_token" {
			t.Errorf("%s avec l'exemple : %d %s, attendu 400 invalid_token", operationID, got.status, got.body)
		}
	}
}
