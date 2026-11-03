package lookup

import "context"

type Candidate struct {
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	Year    int      `json:"year"`
	Cover   string   `json:"cover_url"`
}

type Provider interface {
	Lookup(ctx context.Context, q string) ([]Candidate, error)
}
