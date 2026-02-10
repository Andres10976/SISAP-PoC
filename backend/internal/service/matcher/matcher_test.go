package matcher

import (
	"testing"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
)

func kw(id int, value string) model.Keyword {
	return model.Keyword{ID: id, Value: value}
}

func cert(cn string, sans ...string) *ctlog.ParsedCertificate {
	return &ctlog.ParsedCertificate{
		CommonName: cn,
		SANs:       sans,
	}
}

func TestMatch_NoKeywords(t *testing.T) {
	results := Match(cert("example.com"), nil)
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestMatch_NoMatch(t *testing.T) {
	results := Match(cert("example.com", "www.example.com"), []model.Keyword{kw(1, "foobar")})
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestMatch_CNMatch(t *testing.T) {
	results := Match(cert("example.com"), []model.Keyword{kw(1, "example")})
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].KeywordID != 1 {
		t.Errorf("KeywordID = %d, want 1", results[0].KeywordID)
	}
	if results[0].MatchedDomain != "example.com" {
		t.Errorf("MatchedDomain = %q, want %q", results[0].MatchedDomain, "example.com")
	}
}

func TestMatch_SANMatch(t *testing.T) {
	results := Match(cert("other.com", "www.example.com"), []model.Keyword{kw(1, "example")})
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].MatchedDomain != "www.example.com" {
		t.Errorf("MatchedDomain = %q, want %q", results[0].MatchedDomain, "www.example.com")
	}
}

func TestMatch_CaseInsensitive(t *testing.T) {
	results := Match(cert("EXAMPLE.COM"), []model.Keyword{kw(1, "Example")})
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestMatch_CNPriorityOverSAN(t *testing.T) {
	// Both CN and SAN contain the keyword; CN should win
	results := Match(
		cert("example.com", "example.org"),
		[]model.Keyword{kw(1, "example")},
	)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].MatchedDomain != "example.com" {
		t.Errorf("MatchedDomain = %q, want CN %q", results[0].MatchedDomain, "example.com")
	}
}

func TestMatch_MultipleKeywords(t *testing.T) {
	results := Match(
		cert("example.com", "test.org"),
		[]model.Keyword{kw(1, "example"), kw(2, "test")},
	)
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].KeywordID != 1 {
		t.Errorf("results[0].KeywordID = %d, want 1", results[0].KeywordID)
	}
	if results[1].KeywordID != 2 {
		t.Errorf("results[1].KeywordID = %d, want 2", results[1].KeywordID)
	}
}

func TestMatch_FirstSANWins(t *testing.T) {
	results := Match(
		cert("other.com", "aaa.example.com", "bbb.example.com"),
		[]model.Keyword{kw(1, "example")},
	)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].MatchedDomain != "aaa.example.com" {
		t.Errorf("MatchedDomain = %q, want first SAN %q", results[0].MatchedDomain, "aaa.example.com")
	}
}

func TestMatch_EmptyCN(t *testing.T) {
	results := Match(cert("", "example.com"), []model.Keyword{kw(1, "example")})
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].MatchedDomain != "example.com" {
		t.Errorf("MatchedDomain = %q, want SAN", results[0].MatchedDomain)
	}
}

func TestMatch_EmptySANs(t *testing.T) {
	results := Match(cert("other.com"), []model.Keyword{kw(1, "example")})
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}
