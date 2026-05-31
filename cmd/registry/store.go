package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RegistryStore struct {
	db *sql.DB
}

func NewRegistryStore(db *sql.DB) *RegistryStore {
	return &RegistryStore{db: db}
}

func (s *RegistryStore) validType(t string) (CatalogType, bool) {
	switch CatalogType(t) {
	case CatalogAgents, CatalogSkills, CatalogTools, CatalogModels:
		return CatalogType(t), true
	default:
		return "", false
	}
}

func (s *RegistryStore) List(catType CatalogType) []CatalogEntry {
	rows, err := s.db.Query(`
		SELECT id, type, name, description, metadata, created_at
		FROM catalog_entries WHERE type = $1 ORDER BY name`, string(catType))
	if err != nil {
		log.Printf("List %s: %v", catType, err)
		return []CatalogEntry{}
	}
	defer rows.Close()

	result := make([]CatalogEntry, 0)
	for rows.Next() {
		entry, err := scanCatalogRow(rows)
		if err != nil {
			log.Printf("List scan: %v", err)
			continue
		}
		result = append(result, *entry)
	}
	return result
}

func (s *RegistryStore) Register(catType CatalogType, req RegisterRequest) (*CatalogEntry, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	id := uuid.New().String()
	now := time.Now().UTC()

	// Upsert: if type+name already exists, update it.
	_, err = s.db.Exec(`
		INSERT INTO catalog_entries (id, type, name, description, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (type, name) DO UPDATE SET
			description = EXCLUDED.description,
			metadata = EXCLUDED.metadata`,
		id, string(catType), req.Name, req.Description, metadataJSON, now,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert catalog_entries: %w", err)
	}

	entry := &CatalogEntry{
		Name:        req.Name,
		Type:        catType,
		Version:     version,
		Description: req.Description,
		Metadata:    metadata,
		CreatedAt:   now,
	}
	return entry, nil
}

func (s *RegistryStore) Deregister(catType CatalogType, name string) bool {
	res, err := s.db.Exec(`DELETE FROM catalog_entries WHERE type = $1 AND name = $2`, string(catType), name)
	if err != nil {
		log.Printf("Deregister %s/%s: %v", catType, name, err)
		return false
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false
	}
	return n > 0
}

func (s *RegistryStore) Stats() CatalogStats {
	stats := CatalogStats{}

	rows, err := s.db.Query(`SELECT type, COUNT(*) FROM catalog_entries GROUP BY type`)
	if err != nil {
		log.Printf("Stats catalog: %v", err)
		return stats
	}
	defer rows.Close()

	for rows.Next() {
		var catType string
		var count int
		if err := rows.Scan(&catType, &count); err != nil {
			continue
		}
		stats.Total += count
		switch CatalogType(catType) {
		case CatalogAgents:
			stats.Agents = count
		case CatalogSkills:
			stats.Skills = count
		case CatalogTools:
			stats.Tools = count
		case CatalogModels:
			stats.Models = count
		}
	}

	err = s.db.QueryRow(`SELECT COUNT(*) FROM approval_requests WHERE status = 'pending'`).Scan(&stats.PendingApprovals)
	if err != nil {
		log.Printf("Stats approvals: %v", err)
	}

	err = s.db.QueryRow(`SELECT COUNT(*) FROM package_versions WHERE status = 'published'`).Scan(&stats.PublishedVersions)
	if err != nil {
		log.Printf("Stats versions: %v", err)
	}

	return stats
}

// --- Version tracking ---

func computeDigest(catType, name, version string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s:%d", catType, name, version, time.Now().UnixNano())))
	return fmt.Sprintf("sha256:%x", h[:12])
}

func (s *RegistryStore) ListVersions(catType, name string) []PackageVersion {
	rows, err := s.db.Query(`
		SELECT id, version, author, digest, status, notes, created_at
		FROM package_versions
		WHERE package_type = $1 AND package_name = $2
		ORDER BY created_at DESC`, catType, name)
	if err != nil {
		log.Printf("ListVersions %s/%s: %v", catType, name, err)
		return []PackageVersion{}
	}
	defer rows.Close()

	result := make([]PackageVersion, 0)
	for rows.Next() {
		var pv PackageVersion
		var id string
		if err := rows.Scan(&id, &pv.Version, &pv.Author, &pv.Digest, &pv.Status, &pv.Notes, &pv.CreatedAt); err != nil {
			log.Printf("ListVersions scan: %v", err)
			continue
		}
		result = append(result, pv)
	}
	return result
}

func (s *RegistryStore) AddVersion(catType, name, version, author, notes string) PackageVersion {
	id := uuid.New().String()
	pv := PackageVersion{
		Version:   version,
		CreatedAt: time.Now().UTC(),
		Author:    author,
		Digest:    computeDigest(catType, name, version),
		Status:    "published",
		Notes:     notes,
	}

	_, err := s.db.Exec(`
		INSERT INTO package_versions (id, package_type, package_name, version, author, digest, status, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, catType, name, pv.Version, pv.Author, pv.Digest, pv.Status, pv.Notes, pv.CreatedAt,
	)
	if err != nil {
		log.Printf("AddVersion %s/%s@%s: %v", catType, name, version, err)
	}
	return pv
}

// --- Approval workflow ---

func (s *RegistryStore) ListApprovals(status, catType, author string) []ApprovalRequest {
	query := `SELECT id, package_type, package_name, version, author, status, submitted_at, reviewed_at, reviewed_by, comment, diff
		FROM approval_requests WHERE 1=1`
	var args []interface{}
	idx := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, status)
		idx++
	}
	if catType != "" {
		query += fmt.Sprintf(" AND package_type = $%d", idx)
		args = append(args, catType)
		idx++
	}
	if author != "" {
		query += fmt.Sprintf(" AND author = $%d", idx)
		args = append(args, author)
		idx++
	}
	query += " ORDER BY submitted_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("ListApprovals: %v", err)
		return []ApprovalRequest{}
	}
	defer rows.Close()

	result := make([]ApprovalRequest, 0)
	for rows.Next() {
		a, err := scanApprovalRow(rows)
		if err != nil {
			log.Printf("ListApprovals scan: %v", err)
			continue
		}
		result = append(result, *a)
	}
	return result
}

func (s *RegistryStore) GetApproval(id string) (*ApprovalRequest, bool) {
	row := s.db.QueryRow(`
		SELECT id, package_type, package_name, version, author, status, submitted_at, reviewed_at, reviewed_by, comment, diff
		FROM approval_requests WHERE id = $1`, id)

	var a ApprovalRequest
	var reviewedAt sql.NullTime
	err := row.Scan(&a.ID, &a.PackageType, &a.PackageName, &a.Version, &a.Author,
		&a.Status, &a.SubmittedAt, &reviewedAt, &a.ReviewedBy, &a.Comment, &a.Diff)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("GetApproval %s: %v", id, err)
		return nil, false
	}
	if reviewedAt.Valid {
		a.ReviewedAt = reviewedAt.Time
	}
	return &a, true
}

func (s *RegistryStore) UpdateApproval(id, newStatus, reviewer, comment string) (*ApprovalRequest, error) {
	a, ok := s.GetApproval(id)
	if !ok {
		return nil, fmt.Errorf("approval not found")
	}
	if a.Status != "pending" {
		return nil, fmt.Errorf("approval already %s", a.Status)
	}
	if newStatus != "approved" && newStatus != "rejected" {
		return nil, fmt.Errorf("invalid status: must be approved or rejected")
	}

	now := time.Now().UTC()
	_, err := s.db.Exec(`
		UPDATE approval_requests
		SET status = $1, reviewed_at = $2, reviewed_by = $3, comment = $4
		WHERE id = $5`,
		newStatus, now, reviewer, comment, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update approval: %w", err)
	}

	if newStatus == "approved" {
		_, err = s.db.Exec(`
			UPDATE package_versions SET status = 'approved'
			WHERE package_type = $1 AND package_name = $2 AND version = $3 AND status = 'pending'`,
			a.PackageType, a.PackageName, a.Version,
		)
		if err != nil {
			log.Printf("UpdateApproval: update version status: %v", err)
		}
	}

	a.Status = newStatus
	a.ReviewedAt = now
	a.ReviewedBy = reviewer
	if comment != "" {
		a.Comment = comment
	}
	return a, nil
}

// SubmitVersion creates a new version and optionally an approval request.
func (s *RegistryStore) SubmitVersion(catType, name, version, author, notes string, autoApprove bool) (PackageVersion, *ApprovalRequest, error) {
	status := "published"
	if !autoApprove {
		status = "pending"
	}

	id := uuid.New().String()
	now := time.Now().UTC()
	digest := computeDigest(catType, name, version)

	pv := PackageVersion{
		Version:   version,
		CreatedAt: now,
		Author:    author,
		Digest:    digest,
		Status:    status,
		Notes:     notes,
	}

	_, err := s.db.Exec(`
		INSERT INTO package_versions (id, package_type, package_name, version, author, digest, status, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, catType, name, version, author, digest, status, notes, now,
	)
	if err != nil {
		return pv, nil, fmt.Errorf("insert version: %w", err)
	}

	var approval *ApprovalRequest
	if !autoApprove {
		aprID := fmt.Sprintf("apr-%s", uuid.New().String()[:8])
		approval = &ApprovalRequest{
			ID:          aprID,
			PackageType: catType,
			PackageName: name,
			Version:     version,
			Author:      author,
			Status:      "pending",
			SubmittedAt: now,
		}
		_, err = s.db.Exec(`
			INSERT INTO approval_requests (id, package_type, package_name, version, author, status, submitted_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			approval.ID, approval.PackageType, approval.PackageName, approval.Version,
			approval.Author, approval.Status, approval.SubmittedAt,
		)
		if err != nil {
			return pv, nil, fmt.Errorf("insert approval: %w", err)
		}
	}

	return pv, approval, nil
}

// --- Search ---

func (s *RegistryStore) Search(query, catType, sortBy string) []CatalogEntry {
	sqlQuery := `
		SELECT id, type, name, description, metadata, created_at
		FROM catalog_entries WHERE 1=1`
	var args []interface{}
	idx := 1

	if query != "" {
		sqlQuery += fmt.Sprintf(" AND (LOWER(name) LIKE $%d OR LOWER(description) LIKE $%d)", idx, idx)
		args = append(args, "%"+strings.ToLower(query)+"%")
		idx++
	}
	if catType != "" {
		sqlQuery += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, catType)
		idx++
	}

	switch sortBy {
	case "recent":
		sqlQuery += " ORDER BY created_at DESC"
	default:
		sqlQuery += " ORDER BY name"
	}

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		log.Printf("Search: %v", err)
		return []CatalogEntry{}
	}
	defer rows.Close()

	results := make([]CatalogEntry, 0)
	for rows.Next() {
		entry, err := scanCatalogRow(rows)
		if err != nil {
			log.Printf("Search scan: %v", err)
			continue
		}
		results = append(results, *entry)
	}

	if sortBy == "popular" {
		sort.Slice(results, func(i, j int) bool {
			return results[i].Name < results[j].Name
		})
	}
	return results
}

// --- Scan helpers ---

func scanCatalogRow(rows *sql.Rows) (*CatalogEntry, error) {
	var (
		id          string
		catType     string
		name        string
		description string
		metadata    []byte
		createdAt   time.Time
	)
	err := rows.Scan(&id, &catType, &name, &description, &metadata, &createdAt)
	if err != nil {
		return nil, err
	}

	entry := &CatalogEntry{
		Name:        name,
		Type:        CatalogType(catType),
		Description: description,
		Metadata:    map[string]interface{}{},
		CreatedAt:   createdAt,
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
			return nil, err
		}
	}
	return entry, nil
}

func scanApprovalRow(rows *sql.Rows) (*ApprovalRequest, error) {
	var a ApprovalRequest
	var reviewedAt sql.NullTime
	err := rows.Scan(&a.ID, &a.PackageType, &a.PackageName, &a.Version, &a.Author,
		&a.Status, &a.SubmittedAt, &reviewedAt, &a.ReviewedBy, &a.Comment, &a.Diff)
	if err != nil {
		return nil, err
	}
	if reviewedAt.Valid {
		a.ReviewedAt = reviewedAt.Time
	}
	return &a, nil
}
