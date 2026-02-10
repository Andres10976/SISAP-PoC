package ctlog

import (
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTooShort    = errors.New("leaf input too short")
	ErrUnknownType = errors.New("unknown entry type")
	ErrParseFailed = errors.New("certificate parse failed")
)

// ParsedCertificate holds the fields extracted from a CT log entry
// that are relevant for keyword matching and display.
type ParsedCertificate struct {
	Timestamp  time.Time
	Serial     string
	CommonName string
	SANs       []string
	Issuer     string
	NotBefore  time.Time
	NotAfter   time.Time
}

// ParseLeafInput decodes a MerkleTreeLeaf binary blob into a ParsedCertificate.
// It handles both x509_entry and precert_entry types.
// For precert entries (type 1), the actual certificate is extracted from extraData
// (the entry's extra_data field) because leaf_input only contains the TBS, which
// is not a valid DER certificate.
func ParseLeafInput(data []byte, extraData []byte) (*ParsedCertificate, error) {
	if len(data) < 15 {
		return nil, ErrTooShort
	}

	// Bytes 2-9: timestamp (uint64 big-endian, milliseconds since epoch)
	timestamp := binary.BigEndian.Uint64(data[2:10])

	// Bytes 10-11: entry type
	entryType := binary.BigEndian.Uint16(data[10:12])

	var certDER []byte

	switch entryType {
	case 0: // x509_entry
		certLen := readUint24(data[12:15])
		end := 15 + certLen
		if len(data) < end {
			return nil, ErrTooShort
		}
		certDER = data[15:end]

	case 1: // precert_entry — extract certificate from extra_data
		if len(extraData) < 3 {
			return nil, fmt.Errorf("%w: precert extra_data too short", ErrTooShort)
		}
		certLen := readUint24(extraData[0:3])
		end := 3 + certLen
		if len(extraData) < end {
			return nil, fmt.Errorf("%w: precert extra_data truncated", ErrTooShort)
		}
		certDER = extraData[3:end]

	default:
		return nil, fmt.Errorf("%w: %d", ErrUnknownType, entryType)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
	}

	issuer := cert.Issuer.CommonName
	if issuer == "" && len(cert.Issuer.Organization) > 0 {
		issuer = cert.Issuer.Organization[0]
	}

	return &ParsedCertificate{
		Timestamp:  time.UnixMilli(int64(timestamp)),
		Serial:     cert.SerialNumber.Text(16),
		CommonName: cert.Subject.CommonName,
		SANs:       cert.DNSNames,
		Issuer:     issuer,
		NotBefore:  cert.NotBefore,
		NotAfter:   cert.NotAfter,
	}, nil
}

// readUint24 reads a 3-byte big-endian unsigned integer.
func readUint24(b []byte) int {
	return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}
