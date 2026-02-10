package matcher

import (
	"strings"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
)

// MatchResult pairs a keyword ID with the domain that triggered the match.
type MatchResult struct {
	KeywordID     int
	MatchedDomain string
}

// Match checks a parsed certificate against all keywords.
// Returns one match per keyword (first matching domain wins).
func Match(cert *ctlog.ParsedCertificate, keywords []model.Keyword) []MatchResult {
	var results []MatchResult

	for _, kw := range keywords {
		lower := strings.ToLower(kw.Value)

		// Check Common Name first
		if cert.CommonName != "" && strings.Contains(strings.ToLower(cert.CommonName), lower) {
			results = append(results, MatchResult{
				KeywordID:     kw.ID,
				MatchedDomain: cert.CommonName,
			})
			continue
		}

		// Check each SAN
		for _, san := range cert.SANs {
			if strings.Contains(strings.ToLower(san), lower) {
				results = append(results, MatchResult{
					KeywordID:     kw.ID,
					MatchedDomain: san,
				})
				break
			}
		}
	}

	return results
}
