package catalog

import (
	"context"
	"fmt"
)

import "sampo/internal/domain"

type Stats struct {
	Providers        int `json:"providers"`
	Contents         int `json:"contents"`
	AvailableFiles   int `json:"availableFiles"`
	UnavailableFiles int `json:"unavailableFiles"`
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var stats Stats
	if err := s.db.QueryRowContext(ctx, `SELECT
        (SELECT count(*) FROM providers),
        (SELECT count(*) FROM contents),
        (SELECT count(*) FROM appearances WHERE availability='available'),
        (SELECT count(*) FROM appearances WHERE availability='unavailable')`).Scan(
		&stats.Providers, &stats.Contents, &stats.AvailableFiles, &stats.UnavailableFiles); err != nil {
		return Stats{}, fmt.Errorf("read catalogue stats: %w", err)
	}
	return stats, nil
}

func (s *Store) Search(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT
        c.id, c.algorithm, c.digest_hex, c.byte_length,
        a.id, a.provider_id, a.locator, a.display_name, a.native_identity,
        a.modified_at_ns, a.custody, a.availability, a.continuity, a.observed_at_ns
        FROM contents c JOIN appearances a ON a.content_id=c.id
        WHERE (? = '' OR c.id IN (
            SELECT matching.content_id FROM appearances matching
            WHERE instr(lower(matching.locator), lower(?)) > 0
        ))
        ORDER BY c.id, a.availability DESC, a.locator
        LIMIT ?`, query, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search catalogue: %w", err)
	}
	defer rows.Close()

	var results []domain.SearchResult
	byContent := make(map[string]int)
	for rows.Next() {
		var content domain.Content
		var appearance domain.Appearance
		var modified, observed int64
		if err := rows.Scan(&content.ID, &content.Algorithm, &content.DigestHex, &content.ByteLength,
			&appearance.ID, &appearance.ProviderID, &appearance.Locator, &appearance.DisplayName,
			&appearance.NativeIdentity, &modified, &appearance.Custody, &appearance.Availability,
			&appearance.Continuity, &observed); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		appearance.ContentID = content.ID
		appearance.ByteLength = content.ByteLength
		appearance.ModifiedAt = fromUnixNano(modified)
		appearance.ObservedAt = fromUnixNano(observed)
		index, ok := byContent[content.ID]
		if !ok {
			index = len(results)
			byContent[content.ID] = index
			results = append(results, domain.SearchResult{Content: content})
		}
		results[index].Appearances = append(results[index].Appearances, appearance)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}
	return results, nil
}
