// Package apikeyalias stores human-friendly labels for the SHA-256 api key
// hashes usage events carry, so the dashboard can show "production-key"
// instead of "9f3a…". The shape mirrors CPA-Manager-Plus's model.APIKeyAlias.
package apikeyalias

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// APIKeyAlias pairs a SHA-256 api key hash with a display label. JSON tags are
// camelCase to match the front-end ApiKeyAlias interface.
type APIKeyAlias struct {
	APIKeyHash  string `json:"apiKeyHash"`
	Alias       string `json:"alias"`
	UpdatedAtMS int64  `json:"updatedAtMs"`
}

// Repository owns the api_key_aliases table.
type Repository struct {
	db *sql.DB
}

// New returns a repository bound to the given database handle.
func New(db *sql.DB) Repository {
	return Repository{db: db}
}

// LoadAll returns every alias ordered by label (case-insensitive) then hash.
func (r Repository) LoadAll(ctx context.Context) ([]APIKeyAlias, error) {
	rows, errQuery := r.db.QueryContext(ctx, `select api_key_hash, alias, updated_at_ms
		from api_key_aliases
		order by alias collate nocase, api_key_hash`)
	if errQuery != nil {
		return nil, errQuery
	}
	defer func() {
		_ = rows.Close()
	}()

	aliases := []APIKeyAlias{}
	for rows.Next() {
		var alias APIKeyAlias
		if errScan := rows.Scan(&alias.APIKeyHash, &alias.Alias, &alias.UpdatedAtMS); errScan != nil {
			return nil, errScan
		}
		aliases = append(aliases, alias)
	}
	return aliases, rows.Err()
}

// UpsertMany inserts or updates the given aliases. activeHashes, when non-
// empty, marks which existing aliases may be silently reclaimed when their
// hash is no longer present (orphan cleanup); allowOrphanCleanup gates that
// reclamation. The implementation mirrors CPA-Manager-Plus's repository.
func (r Repository) UpsertMany(ctx context.Context, aliases []APIKeyAlias, activeHashes []string, allowOrphanCleanup bool) error {
	if len(aliases) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	normalizedAliases := make([]APIKeyAlias, 0, len(aliases))
	seenAliases := map[string]string{}
	for _, alias := range aliases {
		normalized, errNorm := normalizeAPIKeyAlias(alias, now)
		if errNorm != nil {
			return errNorm
		}
		aliasKey := normalizeAPIKeyAliasUniqueKey(normalized.Alias)
		if existingHash, ok := seenAliases[aliasKey]; ok && existingHash != normalized.APIKeyHash {
			return errors.New("api key alias already exists")
		}
		seenAliases[aliasKey] = normalized.APIKeyHash
		normalizedAliases = append(normalizedAliases, normalized)
	}

	var activeSet map[string]struct{}
	if len(activeHashes) > 0 {
		activeSet = make(map[string]struct{}, len(activeHashes)+len(normalizedAliases))
		for _, h := range activeHashes {
			hash := strings.ToLower(strings.TrimSpace(h))
			if validAPIKeyHash(hash) {
				activeSet[hash] = struct{}{}
			}
		}
		for _, normalized := range normalizedAliases {
			activeSet[normalized.APIKeyHash] = struct{}{}
		}
	}

	tx, errBegin := r.db.BeginTx(ctx, nil)
	if errBegin != nil {
		return errBegin
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	stmt, errPrepare := tx.PrepareContext(ctx, `insert into api_key_aliases (
		api_key_hash, alias, updated_at_ms
	) values (?, ?, ?)
	on conflict(api_key_hash) do update set
		alias = excluded.alias,
		updated_at_ms = excluded.updated_at_ms`)
	if errPrepare != nil {
		return errPrepare
	}
	defer func() {
		_ = stmt.Close()
	}()

	deleteStmt, errDelete := tx.PrepareContext(ctx, `delete from api_key_aliases where api_key_hash = ?`)
	if errDelete != nil {
		return errDelete
	}
	defer func() {
		_ = deleteStmt.Close()
	}()

	existingRows, errExisting := tx.QueryContext(ctx, `select api_key_hash, alias from api_key_aliases`)
	if errExisting != nil {
		return errExisting
	}
	existingAliases := map[string]string{}
	for existingRows.Next() {
		var apiKeyHash string
		var alias string
		if errScan := existingRows.Scan(&apiKeyHash, &alias); errScan != nil {
			_ = existingRows.Close()
			return errScan
		}
		existingAliases[normalizeAPIKeyAliasUniqueKey(alias)] = apiKeyHash
	}
	if errClose := existingRows.Close(); errClose != nil {
		return errClose
	}
	if errIter := existingRows.Err(); errIter != nil {
		return errIter
	}

	for _, normalized := range normalizedAliases {
		aliasKey := normalizeAPIKeyAliasUniqueKey(normalized.Alias)
		if existingHash, ok := existingAliases[aliasKey]; ok && existingHash != normalized.APIKeyHash {
			if activeSet == nil {
				return errors.New("api key alias already exists")
			}
			if _, isActive := activeSet[existingHash]; isActive {
				return errors.New("api key alias already exists")
			}
			if !allowOrphanCleanup {
				return errors.New("api key alias already exists")
			}
			if _, errExec := deleteStmt.ExecContext(ctx, existingHash); errExec != nil {
				return errExec
			}
			delete(existingAliases, aliasKey)
		}
		if _, errExec := stmt.ExecContext(ctx, normalized.APIKeyHash, normalized.Alias, normalized.UpdatedAtMS); errExec != nil {
			return errExec
		}
		existingAliases[aliasKey] = normalized.APIKeyHash
	}
	if errCommit := tx.Commit(); errCommit != nil {
		return errCommit
	}
	committed = true
	return nil
}

// Delete removes a single alias by its SHA-256 hash.
func (r Repository) Delete(ctx context.Context, apiKeyHash string) error {
	hash := strings.ToLower(strings.TrimSpace(apiKeyHash))
	if !validAPIKeyHash(hash) {
		return errors.New("valid apiKeyHash is required")
	}
	_, err := r.db.ExecContext(ctx, `delete from api_key_aliases where api_key_hash = ?`, hash)
	return err
}

func normalizeAPIKeyAlias(alias APIKeyAlias, now int64) (APIKeyAlias, error) {
	hash := strings.ToLower(strings.TrimSpace(alias.APIKeyHash))
	if !validAPIKeyHash(hash) {
		return APIKeyAlias{}, errors.New("valid apiKeyHash is required")
	}
	label := strings.TrimSpace(alias.Alias)
	if label == "" {
		return APIKeyAlias{}, errors.New("alias is required")
	}
	if len([]rune(label)) > 120 {
		return APIKeyAlias{}, errors.New("alias must be 120 characters or less")
	}
	if alias.UpdatedAtMS <= 0 {
		alias.UpdatedAtMS = now
	}
	alias.APIKeyHash = hash
	alias.Alias = label
	return alias, nil
}

func normalizeAPIKeyAliasUniqueKey(alias string) string {
	return strings.ToLower(strings.TrimSpace(alias))
}

func validAPIKeyHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') {
			continue
		}
		return false
	}
	return true
}