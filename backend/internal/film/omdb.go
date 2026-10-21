package film

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrCatalogUnavailable = errors.New("film : catalogue OMDb indisponible")
	ErrTitleNotFound      = errors.New("film : titre introuvable sur OMDb")
)

type Catalog interface {
	Search(ctx context.Context, query string) ([]Title, error)
	Get(ctx context.Context, imdbID string) (*Title, json.RawMessage, error)
}

type OMDbClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewOMDbClient(baseURL, apiKey string, timeout time.Duration) *OMDbClient {
	return &OMDbClient{
		baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey,
		http: &http.Client{Timeout: timeout},
	}
}

type omdbResponse struct {
	Response string `json:"Response"`
	Error    string `json:"Error"`
	Search   []struct {
		ID     string `json:"imdbID"`
		Title  string `json:"Title"`
		Year   string `json:"Year"`
		Poster string `json:"Poster"`
	} `json:"Search"`
	ID     string `json:"imdbID"`
	Title  string `json:"Title"`
	Year   string `json:"Year"`
	Poster string `json:"Poster"`
	Plot   string `json:"Plot"`
}

func (c *OMDbClient) Search(ctx context.Context, query string) ([]Title, error) {
	values := url.Values{"s": {query}, "type": {"movie"}}
	var result omdbResponse
	if err := c.fetch(ctx, values, &result); err != nil {
		return nil, err
	}
	if result.Response == "False" {
		if strings.EqualFold(result.Error, "Movie not found!") || strings.EqualFold(result.Error, "Too many results.") {
			return []Title{}, nil
		}
		return nil, fmt.Errorf("%w: %s", ErrCatalogUnavailable, result.Error)
	}
	titles := make([]Title, 0, len(result.Search))
	for _, item := range result.Search {
		titles = append(titles, Title{ID: item.ID, Title: item.Title, ReleaseYear: parseYear(item.Year), PosterURL: optionalOMDb(item.Poster)})
	}
	return titles, nil
}

func (c *OMDbClient) Get(ctx context.Context, imdbID string) (*Title, json.RawMessage, error) {
	values := url.Values{"i": {imdbID}, "plot": {"full"}}
	var result omdbResponse
	raw, err := c.fetchRaw(ctx, values)
	if err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, nil, fmt.Errorf("film : décodage de la réponse OMDb : %w", err)
	}
	if result.Response == "False" {
		if strings.EqualFold(result.Error, "Movie not found!") {
			return nil, nil, ErrTitleNotFound
		}
		return nil, nil, fmt.Errorf("%w: %s", ErrCatalogUnavailable, result.Error)
	}
	title := &Title{
		ID: result.ID, Title: result.Title, ReleaseYear: parseYear(result.Year),
		PosterURL: optionalOMDb(result.Poster), Plot: optionalOMDb(result.Plot),
	}
	return title, json.RawMessage(raw), nil
}

func (c *OMDbClient) fetch(ctx context.Context, values url.Values, dst *omdbResponse) error {
	raw, err := c.fetchRaw(ctx, values)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("film : décodage de la réponse OMDb : %w", err)
	}
	return nil
}

func (c *OMDbClient) fetchRaw(ctx context.Context, values url.Values) ([]byte, error) {
	if c.apiKey == "" {
		return nil, ErrCatalogUnavailable
	}
	values.Set("apikey", c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/?"+values.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("film : requête OMDb : %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ErrCatalogUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: HTTP %d", ErrCatalogUnavailable, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: lecture de la réponse", ErrCatalogUnavailable)
	}
	return raw, nil
}

func optionalOMDb(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") {
		return ""
	}
	return value
}

func parseYear(value string) *int16 {
	if len(value) < 4 {
		return nil
	}
	year, err := strconv.ParseInt(value[:4], 10, 16)
	if err != nil {
		return nil
	}
	parsed := int16(year)
	return &parsed
}
