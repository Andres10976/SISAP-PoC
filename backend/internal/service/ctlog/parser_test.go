package ctlog

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"errors"
	"math/big"
	"testing"
	"time"
)

// buildLeaf constructs a minimal MerkleTreeLeaf blob for testing.
// entryType: 0 = x509_entry, 1 = precert_entry.
func buildLeaf(t *testing.T, entryType uint16, certDER []byte, ts uint64) []byte {
	t.Helper()

	var buf []byte

	// 2 bytes: version + leaf type (both 0)
	buf = append(buf, 0, 0)

	// 8 bytes: timestamp
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, ts)
	buf = append(buf, tsBytes...)

	// 2 bytes: entry type
	etBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(etBytes, entryType)
	buf = append(buf, etBytes...)

	switch entryType {
	case 0: // x509_entry: 3-byte length + DER
		lenBytes := []byte{
			byte(len(certDER) >> 16),
			byte(len(certDER) >> 8),
			byte(len(certDER)),
		}
		buf = append(buf, lenBytes...)
		buf = append(buf, certDER...)

	case 1: // precert_entry: 32-byte issuer_key_hash + 3-byte length + TBS
		issuerHash := make([]byte, 32)
		buf = append(buf, issuerHash...)
		lenBytes := []byte{
			byte(len(certDER) >> 16),
			byte(len(certDER) >> 8),
			byte(len(certDER)),
		}
		buf = append(buf, lenBytes...)
		buf = append(buf, certDER...)
	}

	return buf
}

// selfSignedCert generates a self-signed certificate DER for testing.
// If org is non-empty, the issuer will have that organization and no CN.
func selfSignedCert(t *testing.T, cn string, sans []string, org string) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		DNSNames:     sans,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}

	// For the issuer org fallback test, we need the parent cert's Subject
	// to have Organization but no CommonName, since x509.CreateCertificate
	// uses the parent's Subject as the child's Issuer.
	parent := tmpl
	if org != "" {
		parentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("generate parent key: %v", err)
		}
		parentTmpl := &x509.Certificate{
			SerialNumber:          big.NewInt(2),
			Subject:               pkix.Name{Organization: []string{org}},
			NotBefore:             time.Now().Add(-time.Hour),
			NotAfter:              time.Now().Add(time.Hour),
			IsCA:                  true,
			BasicConstraintsValid: true,
		}
		parentDER, err := x509.CreateCertificate(rand.Reader, parentTmpl, parentTmpl, &parentKey.PublicKey, parentKey)
		if err != nil {
			t.Fatalf("create parent certificate: %v", err)
		}
		parentCert, err := x509.ParseCertificate(parentDER)
		if err != nil {
			t.Fatalf("parse parent certificate: %v", err)
		}
		parent = parentCert
		_ = parentKey // parent key signs the child
		der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, &key.PublicKey, parentKey)
		if err != nil {
			t.Fatalf("create certificate: %v", err)
		}
		return der
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return der
}

func TestParseLeafInput_X509Entry(t *testing.T) {
	der := selfSignedCert(t, "example.com", []string{"www.example.com"}, "")
	ts := uint64(1700000000000) // some fixed timestamp
	leaf := buildLeaf(t, 0, der, ts)

	pc, err := ParseLeafInput(leaf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pc.CommonName != "example.com" {
		t.Errorf("CommonName = %q, want %q", pc.CommonName, "example.com")
	}
	if len(pc.SANs) != 1 || pc.SANs[0] != "www.example.com" {
		t.Errorf("SANs = %v, want [www.example.com]", pc.SANs)
	}
	if pc.Timestamp != time.UnixMilli(int64(ts)) {
		t.Errorf("Timestamp = %v, want %v", pc.Timestamp, time.UnixMilli(int64(ts)))
	}
}

func TestParseLeafInput_PrecertEntry(t *testing.T) {
	der := selfSignedCert(t, "precert.example.com", nil, "")
	leaf := buildLeaf(t, 1, der, 1700000000000)

	pc, err := ParseLeafInput(leaf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.CommonName != "precert.example.com" {
		t.Errorf("CommonName = %q, want %q", pc.CommonName, "precert.example.com")
	}
}

func TestParseLeafInput_TooShort(t *testing.T) {
	_, err := ParseLeafInput([]byte{0, 0, 0})
	if !errors.Is(err, ErrTooShort) {
		t.Errorf("err = %v, want ErrTooShort", err)
	}
}

func TestParseLeafInput_UnknownType(t *testing.T) {
	// Build a leaf with entry type 99
	leaf := buildLeaf(t, 99, nil, 1700000000000)
	// Need at least 15 bytes; pad if necessary
	for len(leaf) < 15 {
		leaf = append(leaf, 0)
	}

	_, err := ParseLeafInput(leaf)
	if !errors.Is(err, ErrUnknownType) {
		t.Errorf("err = %v, want ErrUnknownType", err)
	}
}

func TestParseLeafInput_TruncatedX509(t *testing.T) {
	// Header says cert is 1000 bytes, but we only supply 5
	leaf := buildLeaf(t, 0, make([]byte, 5), 1700000000000)
	// Overwrite the length field to claim 1000 bytes
	leaf[12] = 0
	leaf[13] = 3
	leaf[14] = 0xe8 // 1000

	_, err := ParseLeafInput(leaf)
	if !errors.Is(err, ErrTooShort) {
		t.Errorf("err = %v, want ErrTooShort", err)
	}
}

func TestParseLeafInput_InvalidDER(t *testing.T) {
	leaf := buildLeaf(t, 0, []byte{0xDE, 0xAD, 0xBE, 0xEF}, 1700000000000)

	_, err := ParseLeafInput(leaf)
	if !errors.Is(err, ErrParseFailed) {
		t.Errorf("err = %v, want ErrParseFailed", err)
	}
}

func TestParseLeafInput_IssuerOrgFallback(t *testing.T) {
	// Create a cert where issuer CN is empty but org is set
	der := selfSignedCert(t, "test.com", nil, "My Org")

	leaf := buildLeaf(t, 0, der, 1700000000000)

	pc, err := ParseLeafInput(leaf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The issuer should be either the CN (from self-signed) or the org
	// Since selfSignedCert sets issuer org when org != "", the issuer.CN will be empty
	// and it should fall back to the organization
	if pc.Issuer != "My Org" {
		t.Errorf("Issuer = %q, want %q", pc.Issuer, "My Org")
	}
}
