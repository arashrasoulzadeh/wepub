package blob

import "strings"

import "testing"

func TestPath(t *testing.T) {
	p, err := Path("/data", "ab12"+strings.Repeat("0",60))
	if err != nil { t.Fatal(err) }
	if p != "/data/blobs/ab/12/ab12"+strings.Repeat("0",60) { t.Fatalf("got %s",p) }
}
func TestRejects(t *testing.T) {
	if _, err := Path("/data", "../etc"); err == nil { t.Fatal("expected err") }
	if _, err := Path("/data", "zz"); err == nil { t.Fatal("expected err") }
}
