package film

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOMDbSearchMapsTitlesAndMissingPosters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("apikey") != "test-key" || r.URL.Query().Get("s") != "matrix" {
			t.Errorf("query OMDb inattendue : %v", r.URL.Query())
		}
		_, _ = w.Write([]byte(`{"Response":"True","Search":[{"imdbID":"tt0133093","Title":"The Matrix","Year":"1999","Poster":"N/A"}]}`))
	}))
	defer server.Close()

	titles, err := NewOMDbClient(server.URL, "test-key", 0).Search(context.Background(), "matrix")
	if err != nil {
		t.Fatal(err)
	}
	if len(titles) != 1 || titles[0].ID != "tt0133093" || titles[0].Title != "The Matrix" || titles[0].PosterURL != "" {
		t.Fatalf("résultats = %+v", titles)
	}
	if titles[0].ReleaseYear == nil || *titles[0].ReleaseYear != 1999 {
		t.Fatalf("année = %v", titles[0].ReleaseYear)
	}
}

func TestOMDbGetReturnsPosterAndRawJSON(t *testing.T) {
	const response = `{"Response":"True","imdbID":"tt0133093","Title":"The Matrix","Year":"1999","Poster":"https://m.media-amazon.com/poster.jpg","Plot":"A hacker discovers reality."}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("i") != "tt0133093" {
			t.Errorf("identifiant IMDb = %q", r.URL.Query().Get("i"))
		}
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	title, raw, err := NewOMDbClient(server.URL, "test-key", 0).Get(context.Background(), "tt0133093")
	if err != nil {
		t.Fatal(err)
	}
	if title.PosterURL != "https://m.media-amazon.com/poster.jpg" || title.Plot == "" {
		t.Fatalf("détail = %+v", title)
	}
	if !json.Valid(raw) || string(raw) != response {
		t.Fatalf("réponse brute = %s", raw)
	}
}

func TestOMDbGetMapsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Movie not found!"}`))
	}))
	defer server.Close()

	_, _, err := NewOMDbClient(server.URL, "test-key", 0).Get(context.Background(), "tt0000000")
	if err != ErrTitleNotFound {
		t.Fatalf("erreur = %v, attendu %v", err, ErrTitleNotFound)
	}
}
