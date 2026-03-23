package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB 数据库实例
type DB struct {
	db *sql.DB
}

// ProjectKey 项目密钥
type ProjectKey struct {
	ID        int64
	ProjectID string
	Key       string
	RateLimit int
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AnthropicKey Anthropic API Key
type AnthropicKey struct {
	ID        int64
	Name      string
	Key       string
	Enabled   bool
	UsageCount int64
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

// ModelConfig 模型配置
type ModelConfig struct {
	ID       int64
	ModelID  string
	Name     string
	Weight   int
	Priority int
	Enabled  bool
}

// New 初始化数据库
func New(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	dbPath := filepath.Join(dataDir, "moderation.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	s := &DB{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return s, nil
}

func (s *DB) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS project_keys (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id TEXT NOT NULL,
			key        TEXT NOT NULL UNIQUE,
			rate_limit INTEGER NOT NULL DEFAULT 60,
			enabled    INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS anthropic_keys (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			name         TEXT NOT NULL,
			key          TEXT NOT NULL UNIQUE,
			enabled      INTEGER NOT NULL DEFAULT 1,
			usage_count  INTEGER NOT NULL DEFAULT 0,
			last_used_at DATETIME,
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS model_configs (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			model_id TEXT NOT NULL UNIQUE,
			name     TEXT NOT NULL,
			weight   INTEGER NOT NULL DEFAULT 50,
			priority INTEGER NOT NULL DEFAULT 1,
			enabled  INTEGER NOT NULL DEFAULT 1
		);
	`)
	return err
}

// ── Project Keys ──────────────────────────────────────────

func (s *DB) ListProjectKeys() ([]*ProjectKey, error) {
	rows, err := s.db.Query(`SELECT id, project_id, key, rate_limit, enabled, created_at, updated_at FROM project_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*ProjectKey
	for rows.Next() {
		k := &ProjectKey{}
		var enabled int
		err := rows.Scan(&k.ID, &k.ProjectID, &k.Key, &k.RateLimit, &enabled, &k.CreatedAt, &k.UpdatedAt)
		if err != nil {
			return nil, err
		}
		k.Enabled = enabled == 1
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *DB) AddProjectKey(projectID, key string, rateLimit int) (*ProjectKey, error) {
	now := time.Now()
	result, err := s.db.Exec(
		`INSERT INTO project_keys (project_id, key, rate_limit, enabled, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`,
		projectID, key, rateLimit, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &ProjectKey{ID: id, ProjectID: projectID, Key: key, RateLimit: rateLimit, Enabled: true, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *DB) UpdateProjectKey(currentKey string, projectID *string, newKey *string, enabled *bool, rateLimit *int) error {
	if projectID != nil {
		_, err := s.db.Exec(`UPDATE project_keys SET project_id=?, updated_at=? WHERE key=?`, *projectID, time.Now(), currentKey)
		if err != nil {
			return err
		}
	}
	if newKey != nil {
		_, err := s.db.Exec(`UPDATE project_keys SET key=?, updated_at=? WHERE key=?`, *newKey, time.Now(), currentKey)
		if err != nil {
			return err
		}
		currentKey = *newKey
	}
	if enabled != nil {
		e := 0
		if *enabled {
			e = 1
		}
		_, err := s.db.Exec(`UPDATE project_keys SET enabled=?, updated_at=? WHERE key=?`, e, time.Now(), currentKey)
		if err != nil {
			return err
		}
	}
	if rateLimit != nil {
		_, err := s.db.Exec(`UPDATE project_keys SET rate_limit=?, updated_at=? WHERE key=?`, *rateLimit, time.Now(), currentKey)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DB) DeleteProjectKey(key string) error {
	_, err := s.db.Exec(`DELETE FROM project_keys WHERE key=?`, key)
	return err
}

func (s *DB) GetEnabledProjectKeys() ([]string, error) {
	rows, err := s.db.Query(`SELECT key FROM project_keys WHERE enabled=1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		rows.Scan(&k)
		keys = append(keys, k)
	}
	return keys, nil
}

// ── Anthropic Keys ────────────────────────────────────────

func (s *DB) ListAnthropicKeys() ([]*AnthropicKey, error) {
	rows, err := s.db.Query(`SELECT id, name, key, enabled, usage_count, last_used_at, created_at FROM anthropic_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*AnthropicKey
	for rows.Next() {
		k := &AnthropicKey{}
		var enabled int
		err := rows.Scan(&k.ID, &k.Name, &k.Key, &enabled, &k.UsageCount, &k.LastUsedAt, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		k.Enabled = enabled == 1
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *DB) AddAnthropicKey(name, key string) (*AnthropicKey, error) {
	now := time.Now()
	result, err := s.db.Exec(
		`INSERT INTO anthropic_keys (name, key, enabled, created_at) VALUES (?, ?, 1, ?)`,
		name, key, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &AnthropicKey{ID: id, Name: name, Key: key, Enabled: true, CreatedAt: now}, nil
}

func (s *DB) UpdateAnthropicKey(id int64, enabled bool) error {
	e := 0
	if enabled {
		e = 1
	}
	_, err := s.db.Exec(`UPDATE anthropic_keys SET enabled=? WHERE id=?`, e, id)
	return err
}

func (s *DB) DeleteAnthropicKey(id int64) error {
	_, err := s.db.Exec(`DELETE FROM anthropic_keys WHERE id=?`, id)
	return err
}

func (s *DB) IncrAnthropicKeyUsage(id int64) error {
	_, err := s.db.Exec(`UPDATE anthropic_keys SET usage_count=usage_count+1, last_used_at=? WHERE id=?`, time.Now(), id)
	return err
}

func (s *DB) GetEnabledAnthropicKeys() ([]*AnthropicKey, error) {
	rows, err := s.db.Query(`SELECT id, name, key, enabled, usage_count, last_used_at, created_at FROM anthropic_keys WHERE enabled=1 ORDER BY usage_count ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*AnthropicKey
	for rows.Next() {
		k := &AnthropicKey{}
		var enabled int
		rows.Scan(&k.ID, &k.Name, &k.Key, &enabled, &k.UsageCount, &k.LastUsedAt, &k.CreatedAt)
		k.Enabled = enabled == 1
		keys = append(keys, k)
	}
	return keys, nil
}

// ── Model Configs ─────────────────────────────────────────

func (s *DB) ListModels() ([]*ModelConfig, error) {
	rows, err := s.db.Query(`SELECT id, model_id, name, weight, priority, enabled FROM model_configs ORDER BY priority ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*ModelConfig
	for rows.Next() {
		m := &ModelConfig{}
		var enabled int
		rows.Scan(&m.ID, &m.ModelID, &m.Name, &m.Weight, &m.Priority, &enabled)
		m.Enabled = enabled == 1
		models = append(models, m)
	}
	return models, nil
}

func (s *DB) UpsertModel(modelID, name string, weight, priority int, enabled bool) error {
	e := 0
	if enabled {
		e = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO model_configs (model_id, name, weight, priority, enabled)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(model_id) DO UPDATE SET name=excluded.name, weight=excluded.weight, priority=excluded.priority, enabled=excluded.enabled
	`, modelID, name, weight, priority, e)
	return err
}

func (s *DB) UpdateModel(id int64, weight *int, priority *int, enabled *bool) error {
	if weight != nil {
		s.db.Exec(`UPDATE model_configs SET weight=? WHERE id=?`, *weight, id)
	}
	if priority != nil {
		s.db.Exec(`UPDATE model_configs SET priority=? WHERE id=?`, *priority, id)
	}
	if enabled != nil {
		e := 0
		if *enabled {
			e = 1
		}
		s.db.Exec(`UPDATE model_configs SET enabled=? WHERE id=?`, e, id)
	}
	return nil
}

func (s *DB) DeleteModel(id int64) error {
	_, err := s.db.Exec(`DELETE FROM model_configs WHERE id=?`, id)
	return err
}

func (s *DB) Close() error {
	return s.db.Close()
}
