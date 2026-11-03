package lookup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type OpenLibrary struct{ Client *http.Client }

func (o *OpenLibrary) Lookup(ctx context.Context, q string) ([]Candidate, error) {
	if o.Client == nil { o.Client = http.DefaultClient }
	resp, err := o.Client.Get("https://openlibrary.org/search.json?q=" + q)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	var raw struct {
		Docs []struct {
			Title      string   `json:"title"`
			AuthorName []string `json:"author_name"`
			FirstYear  int      `json:"first_publish_year"`
			CoverI     int      `json:"cover_i"`
		} `json:"docs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { return nil, err }
	var out []Candidate
	for _, d := range raw.Docs {
		c := Candidate{Title: d.Title, Authors: d.AuthorName, Year: d.FirstYear}
		if d.CoverI > 0 {
			c.Cover = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg", d.CoverI)
		}
		out = append(out, c)
	}
	return out, nil
}
