package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
)

type CertificateRepository struct {
	pool *pgxpool.Pool
}

func NewCertificateRepository(pool *pgxpool.Pool) *CertificateRepository {
	return &CertificateRepository{pool: pool}
}

func (r *CertificateRepository) Create(ctx context.Context, cert *model.MatchedCertificate) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO matched_certificates
			(serial_number, common_name, sans, issuer, not_before, not_after,
			 keyword_id, matched_domain, ct_log_index)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (serial_number, keyword_id) DO NOTHING`,
		cert.SerialNumber, cert.CommonName, cert.SANs, cert.Issuer,
		cert.NotBefore, cert.NotAfter, cert.KeywordID, cert.MatchedDomain,
		cert.CTLogIndex,
	)
	return err
}

func (r *CertificateRepository) ListPaginated(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM matched_certificates`
	args := []any{}
	argIdx := 1

	if keywordID > 0 {
		countQuery += fmt.Sprintf(` WHERE keyword_id = $%d`, argIdx)
		args = append(args, keywordID)
		argIdx++
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page
	dataQuery := `SELECT mc.id, mc.serial_number, mc.common_name, mc.sans, mc.issuer,
		mc.not_before, mc.not_after, mc.keyword_id, k.value, mc.matched_domain,
		mc.ct_log_index, mc.discovered_at
	FROM matched_certificates mc
	JOIN keywords k ON k.id = mc.keyword_id`

	dataArgs := []any{}
	dataArgIdx := 1

	if keywordID > 0 {
		dataQuery += fmt.Sprintf(` WHERE mc.keyword_id = $%d`, dataArgIdx)
		dataArgs = append(dataArgs, keywordID)
		dataArgIdx++
	}

	dataQuery += ` ORDER BY mc.discovered_at DESC`
	dataQuery += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, dataArgIdx, dataArgIdx+1)
	dataArgs = append(dataArgs, perPage, (page-1)*perPage)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var certs []model.MatchedCertificate
	for rows.Next() {
		var c model.MatchedCertificate
		if err := rows.Scan(
			&c.ID, &c.SerialNumber, &c.CommonName, &c.SANs, &c.Issuer,
			&c.NotBefore, &c.NotAfter, &c.KeywordID, &c.KeywordValue,
			&c.MatchedDomain, &c.CTLogIndex, &c.DiscoveredAt,
		); err != nil {
			return nil, 0, err
		}
		certs = append(certs, c)
	}
	return certs, total, rows.Err()
}

func (r *CertificateRepository) ExportAll(ctx context.Context) ([]model.MatchedCertificate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT mc.id, mc.serial_number, mc.common_name, mc.sans, mc.issuer,
			mc.not_before, mc.not_after, mc.keyword_id, k.value, mc.matched_domain,
			mc.ct_log_index, mc.discovered_at
		FROM matched_certificates mc
		JOIN keywords k ON k.id = mc.keyword_id
		ORDER BY mc.discovered_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []model.MatchedCertificate
	for rows.Next() {
		var c model.MatchedCertificate
		if err := rows.Scan(
			&c.ID, &c.SerialNumber, &c.CommonName, &c.SANs, &c.Issuer,
			&c.NotBefore, &c.NotAfter, &c.KeywordID, &c.KeywordValue,
			&c.MatchedDomain, &c.CTLogIndex, &c.DiscoveredAt,
		); err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, rows.Err()
}
