package search

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krasis/krasis/pkg/types"
)

type SearchResult struct {
	Type       string    `json:"type"`
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Highlights []string  `json:"highlights"`
	Score      float64   `json:"score"`
	UpdatedAt  types.NullTime `json:"updated_at"`
}

type SearchRepository struct {
	pool *pgxpool.Pool
}

func NewSearchRepository(pool *pgxpool.Pool) *SearchRepository {
	return &SearchRepository{pool: pool}
}

func (r *SearchRepository) Search(ctx context.Context, query, searchType string, page, size int) ([]*SearchResult, int64, error) {
	if query == "" {
		return nil, 0, nil
	}

	// Sanitize query for tsquery: replace spaces with "&" for AND search
	tsQuery := strings.ReplaceAll(strings.TrimSpace(query), " ", " & ")

	if searchType == "files" {
		return []*SearchResult{}, 0, nil // Files search not yet implemented
	}

	offset := (page - 1) * size

	// Try hybrid FTS first, fallback to simple FTS if zhparser not available
	results, total, err := r.searchHybrid(ctx, tsQuery, page, size, offset)
	if err != nil {
		// If Chinese FTS fails (zhparser not installed), fallback to simple FTS
		return r.searchSimple(ctx, tsQuery, page, size, offset)
	}

	if total == 0 {
		// Fallback: trigram fuzzy search
		return r.searchByTrigram(ctx, query, page, size)
	}

	return results, total, nil
}

// searchHybrid performs hybrid search with Chinese and simple FTS
func (r *SearchRepository) searchHybrid(ctx context.Context, tsQuery string, page, size, offset int) ([]*SearchResult, int64, error) {
	args := []interface{}{tsQuery, tsQuery, size, offset}

	queryStr := `
		WITH zh_results AS (
			SELECT n.id, n.title,
				ts_headline('chinese_zh', n.content, to_tsquery('chinese_zh', $1),
					'StartSel=<em> StopSel=</em> MaxWords=35 MinWords=10') AS highlights,
				ts_rank(n.search_vector_zh, to_tsquery('chinese_zh', $1)) AS score,
				n.updated_at
			FROM notes n
			WHERE n.is_deleted = false
			  AND n.search_vector_zh @@ to_tsquery('chinese_zh', $1)
		),
		simple_results AS (
			SELECT n.id, n.title,
				ts_headline('simple', n.content, to_tsquery('simple', $2),
					'StartSel=<em> StopSel=</em> MaxWords=35 MinWords=10') AS highlights,
				ts_rank(n.search_vector, to_tsquery('simple', $2)) AS score,
				n.updated_at
			FROM notes n
			WHERE n.is_deleted = false
			  AND n.search_vector @@ to_tsquery('simple', $2)
		),
		merged AS (
			SELECT id, title, highlights, score, updated_at FROM zh_results
			UNION ALL
			SELECT id, title, highlights, score, updated_at FROM simple_results
		),
		ranked AS (
			SELECT DISTINCT ON (id) id, title, score, updated_at,
				MAX(highlights) AS highlights
			FROM merged
			GROUP BY id, title, score, updated_at
			ORDER BY id, score DESC
		)
		SELECT id, title, highlights, score, updated_at
		FROM ranked
		ORDER BY score DESC
		LIMIT $3 OFFSET $4
	`

	var total int64
	countQuery := `
		SELECT COUNT(DISTINCT n.id)
		FROM notes n
		WHERE n.is_deleted = false
		  AND (
			n.search_vector_zh @@ to_tsquery('chinese_zh', $1)
			OR n.search_vector @@ to_tsquery('simple', $1)
		  )
	`
	if err := r.pool.QueryRow(ctx, countQuery, tsQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, queryStr, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var sr SearchResult
		var highlights string
		sr.Type = "note"
		if err := rows.Scan(&sr.ID, &sr.Title, &highlights, &sr.Score, &sr.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if highlights != "" {
			sr.Highlights = []string{highlights}
		}
		results = append(results, &sr)
	}

	return results, total, nil
}

// searchSimple performs simple FTS search without Chinese support
func (r *SearchRepository) searchSimple(ctx context.Context, tsQuery string, page, size, offset int) ([]*SearchResult, int64, error) {
	queryStr := `
		SELECT n.id, n.title,
			ts_headline('simple', n.content, to_tsquery('simple', $1),
				'StartSel=<em> StopSel=</em> MaxWords=35 MinWords=10') AS highlights,
			ts_rank(n.search_vector, to_tsquery('simple', $1)) AS score,
			n.updated_at
		FROM notes n
		WHERE n.is_deleted = false
		  AND n.search_vector @@ to_tsquery('simple', $1)
		ORDER BY score DESC
		LIMIT $2 OFFSET $3
	`

	var total int64
	countQuery := `
		SELECT COUNT(*)
		FROM notes n
		WHERE n.is_deleted = false
		  AND n.search_vector @@ to_tsquery('simple', $1)
	`
	if err := r.pool.QueryRow(ctx, countQuery, tsQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, queryStr, tsQuery, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var sr SearchResult
		var highlights string
		sr.Type = "note"
		if err := rows.Scan(&sr.ID, &sr.Title, &highlights, &sr.Score, &sr.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if highlights != "" {
			sr.Highlights = []string{highlights}
		}
		results = append(results, &sr)
	}

	return results, total, nil
}

// SearchByKeyword searches notes for a specific user using hybrid FTS (Chinese + English + trigram fallback)
func (r *SearchRepository) SearchByKeyword(ctx context.Context, query, userID string, topK int) ([]*SearchResult, error) {
	if query == "" || userID == "" {
		return []*SearchResult{}, nil
	}

	tsQuery := strings.ReplaceAll(strings.TrimSpace(query), " ", " & ")

	// Hybrid FTS with user filter
	queryStr := `
		WITH zh_results AS (
			SELECT n.id, n.title,
				ts_headline('chinese_zh', n.content, to_tsquery('chinese_zh', $1),
					'StartSel=<em> StopSel=</em> MaxWords=50 MinWords=15') AS highlights,
				ts_rank(n.search_vector_zh, to_tsquery('chinese_zh', $1)) AS score,
				n.updated_at
			FROM notes n
			WHERE n.is_deleted = false AND n.owner_id = $2
			  AND n.search_vector_zh @@ to_tsquery('chinese_zh', $1)
		),
		simple_results AS (
			SELECT n.id, n.title,
				ts_headline('simple', n.content, to_tsquery('simple', $1),
					'StartSel=<em> StopSel=</em> MaxWords=50 MinWords=15') AS highlights,
				ts_rank(n.search_vector, to_tsquery('simple', $1)) AS score,
				n.updated_at
			FROM notes n
			WHERE n.is_deleted = false AND n.owner_id = $2
			  AND n.search_vector @@ to_tsquery('simple', $1)
		),
		merged AS (
			SELECT id, title, highlights, score, updated_at FROM zh_results
			UNION ALL
			SELECT id, title, highlights, score, updated_at FROM simple_results
		),
		ranked AS (
			SELECT DISTINCT ON (id) id, title, score, updated_at,
				MAX(highlights) AS highlights
			FROM merged
			GROUP BY id, title, score, updated_at
			ORDER BY id, score DESC
		)
		SELECT id, title, highlights, score, updated_at
		FROM ranked
		ORDER BY score DESC
		LIMIT $3
	`

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, queryStr, tsQuery, uid, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var sr SearchResult
		var highlights string
		sr.Type = "note"
		if err := rows.Scan(&sr.ID, &sr.Title, &highlights, &sr.Score, &sr.UpdatedAt); err != nil {
			return nil, err
		}
		if highlights != "" {
			sr.Highlights = []string{highlights}
		}
		results = append(results, &sr)
	}

	// Fallback to trigram if no FTS results
	if len(results) == 0 {
		return r.searchByKeywordTrigram(ctx, query, uid, topK)
	}

	return results, nil
}

// searchByKeywordTrigram is a user-scoped trigram fallback for SearchByKeyword
func (r *SearchRepository) searchByKeywordTrigram(ctx context.Context, query string, userID uuid.UUID, topK int) ([]*SearchResult, error) {
	queryStr := `
		SELECT n.id, n.title,
			n.title AS highlights,
			similarity(n.title, $1) AS score,
			n.updated_at
		FROM notes n
		WHERE n.is_deleted = false AND n.owner_id = $2
		  AND (n.title % $1 OR n.content % $1)
		ORDER BY score DESC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, queryStr, query, userID, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var sr SearchResult
		var hl string
		sr.Type = "note"
		if err := rows.Scan(&sr.ID, &sr.Title, &hl, &sr.Score, &sr.UpdatedAt); err != nil {
			return nil, err
		}
		if hl != "" {
			sr.Highlights = []string{hl}
		}
		results = append(results, &sr)
	}

	return results, nil
}

// searchByTrigram performs fuzzy matching using pg_trgm when FTS returns no results
func (r *SearchRepository) searchByTrigram(ctx context.Context, query string, page, size int) ([]*SearchResult, int64, error) {
	offset := (page - 1) * size

	// Use % for similarity matching
	queryStr := `
		SELECT n.id, n.title,
			n.title AS highlights,
			similarity(n.title, $1) AS score,
			n.updated_at
		FROM notes n
		WHERE n.is_deleted = false
		  AND (n.title % $1 OR n.content % $1)
		ORDER BY score DESC
		LIMIT $2 OFFSET $3
	`

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM notes n
		WHERE n.is_deleted = false AND (n.title % $1 OR n.content % $1)
	`
	if err := r.pool.QueryRow(ctx, countQuery, query).Scan(&total); err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*SearchResult{}, 0, nil
	}

	rows, err := r.pool.Query(ctx, queryStr, query, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var sr SearchResult
		var hl string
		sr.Type = "note"
		if err := rows.Scan(&sr.ID, &sr.Title, &hl, &sr.Score, &sr.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if hl != "" {
			sr.Highlights = []string{hl}
		}
		results = append(results, &sr)
	}

	return results, total, nil
}
