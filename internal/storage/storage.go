package storage

import (
	"database/sql"
	"errors"
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
	ID         int64
	Name       string
	Key        string
	Enabled    bool
	Status     string     // healthy | unhealthy | unknown
	UsageCount int64
	LastUsedAt *time.Time
	CheckedAt  *time.Time
	CreatedAt  time.Time
}

// ProviderKey OpenAI / Grok API Key
type ProviderKey struct {
	ID         int64
	Provider   string // openai | grok
	Name       string
	Key        string
	Enabled    bool
	Status     string     // healthy | unhealthy | unknown
	UsageCount int64
	LastUsedAt *time.Time
	CheckedAt  *time.Time
	CreatedAt  time.Time
}

// ModelConfig 模型配置
type ModelConfig struct {
	ID       int64
	ModelID  string
	Name     string
	Provider string // anthropic | openai | grok
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
	s.migrateV2() // 追加列，忽略错误（列已存在时会报错）

	return s, nil
}

// migrateV2 为已有表追加新列（幂等，失败静默忽略）
func (s *DB) migrateV2() {
	s.db.Exec(`ALTER TABLE model_configs ADD COLUMN provider TEXT NOT NULL DEFAULT ''`)
	s.db.Exec(`ALTER TABLE anthropic_keys ADD COLUMN status TEXT NOT NULL DEFAULT 'unknown'`)
	s.db.Exec(`ALTER TABLE anthropic_keys ADD COLUMN checked_at DATETIME`)
	s.db.Exec(`ALTER TABLE provider_keys ADD COLUMN status TEXT NOT NULL DEFAULT 'unknown'`)
	s.db.Exec(`ALTER TABLE provider_keys ADD COLUMN checked_at DATETIME`)
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
			status       TEXT NOT NULL DEFAULT 'unknown',
			usage_count  INTEGER NOT NULL DEFAULT 0,
			last_used_at DATETIME,
			checked_at   DATETIME,
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS provider_keys (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			provider     TEXT NOT NULL,
			name         TEXT NOT NULL,
			key          TEXT NOT NULL UNIQUE,
			enabled      INTEGER NOT NULL DEFAULT 1,
			status       TEXT NOT NULL DEFAULT 'unknown',
			usage_count  INTEGER NOT NULL DEFAULT 0,
			last_used_at DATETIME,
			checked_at   DATETIME,
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS model_configs (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			model_id TEXT NOT NULL UNIQUE,
			name     TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT '',
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

func (s *DB) GetEnabledProjectKey(key string) (*ProjectKey, error) {
	row := s.db.QueryRow(`
		SELECT id, project_id, key, rate_limit, enabled, created_at, updated_at
		FROM project_keys
		WHERE key=? AND enabled=1
		LIMIT 1
	`, key)

	projectKey := &ProjectKey{}
	var enabled int
	err := row.Scan(
		&projectKey.ID,
		&projectKey.ProjectID,
		&projectKey.Key,
		&projectKey.RateLimit,
		&enabled,
		&projectKey.CreatedAt,
		&projectKey.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	projectKey.Enabled = enabled == 1
	return projectKey, nil
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
	rows, err := s.db.Query(`SELECT id, name, key, enabled, status, usage_count, last_used_at, checked_at, created_at FROM anthropic_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*AnthropicKey
	for rows.Next() {
		k := &AnthropicKey{}
		var enabled int
		err := rows.Scan(&k.ID, &k.Name, &k.Key, &enabled, &k.Status, &k.UsageCount, &k.LastUsedAt, &k.CheckedAt, &k.CreatedAt)
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
	// 健康的 key 优先，其次 unknown，最后 unhealthy；同等状态按用量最少优先
	rows, err := s.db.Query(`
		SELECT id, name, key, enabled, status, usage_count, last_used_at, checked_at, created_at
		FROM anthropic_keys WHERE enabled=1
		ORDER BY CASE status WHEN 'healthy' THEN 0 WHEN 'unknown' THEN 1 ELSE 2 END, usage_count ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*AnthropicKey
	for rows.Next() {
		k := &AnthropicKey{}
		var enabled int
		rows.Scan(&k.ID, &k.Name, &k.Key, &enabled, &k.Status, &k.UsageCount, &k.LastUsedAt, &k.CheckedAt, &k.CreatedAt)
		k.Enabled = enabled == 1
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *DB) UpdateAnthropicKeyStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE anthropic_keys SET status=?, checked_at=? WHERE id=?`, status, time.Now(), id)
	return err
}

// ── Provider Keys (OpenAI / Grok) ─────────────────────────

func (s *DB) ListProviderKeys(provider string) ([]*ProviderKey, error) {
	rows, err := s.db.Query(`SELECT id, provider, name, key, enabled, status, usage_count, last_used_at, checked_at, created_at FROM provider_keys WHERE provider=? ORDER BY created_at DESC`, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []*ProviderKey
	for rows.Next() {
		k := &ProviderKey{}
		var enabled int
		rows.Scan(&k.ID, &k.Provider, &k.Name, &k.Key, &enabled, &k.Status, &k.UsageCount, &k.LastUsedAt, &k.CheckedAt, &k.CreatedAt)
		k.Enabled = enabled == 1
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *DB) GetEnabledProviderKeys(provider string) ([]*ProviderKey, error) {
	rows, err := s.db.Query(`
		SELECT id, provider, name, key, enabled, status, usage_count, last_used_at, checked_at, created_at
		FROM provider_keys WHERE provider=? AND enabled=1
		ORDER BY CASE status WHEN 'healthy' THEN 0 WHEN 'unknown' THEN 1 ELSE 2 END, usage_count ASC`, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []*ProviderKey
	for rows.Next() {
		k := &ProviderKey{}
		var enabled int
		rows.Scan(&k.ID, &k.Provider, &k.Name, &k.Key, &enabled, &k.Status, &k.UsageCount, &k.LastUsedAt, &k.CheckedAt, &k.CreatedAt)
		k.Enabled = enabled == 1
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *DB) UpdateProviderKeyStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE provider_keys SET status=?, checked_at=? WHERE id=?`, status, time.Now(), id)
	return err
}

func (s *DB) AddProviderKey(provider, name, key string) (*ProviderKey, error) {
	now := time.Now()
	result, err := s.db.Exec(`INSERT INTO provider_keys (provider, name, key, enabled, created_at) VALUES (?, ?, ?, 1, ?)`, provider, name, key, now)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &ProviderKey{ID: id, Provider: provider, Name: name, Key: key, Enabled: true, CreatedAt: now}, nil
}

func (s *DB) UpdateProviderKey(id int64, enabled bool) error {
	e := 0
	if enabled {
		e = 1
	}
	_, err := s.db.Exec(`UPDATE provider_keys SET enabled=? WHERE id=?`, e, id)
	return err
}

func (s *DB) DeleteProviderKey(id int64) error {
	_, err := s.db.Exec(`DELETE FROM provider_keys WHERE id=?`, id)
	return err
}

func (s *DB) IncrProviderKeyUsage(id int64) {
	s.db.Exec(`UPDATE provider_keys SET usage_count=usage_count+1, last_used_at=? WHERE id=?`, time.Now(), id)
}

// ── Model Configs ─────────────────────────────────────────

func (s *DB) ListModels() ([]*ModelConfig, error) {
	rows, err := s.db.Query(`SELECT id, model_id, name, provider, weight, priority, enabled FROM model_configs ORDER BY priority ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*ModelConfig
	for rows.Next() {
		m := &ModelConfig{}
		var enabled int
		rows.Scan(&m.ID, &m.ModelID, &m.Name, &m.Provider, &m.Weight, &m.Priority, &enabled)
		m.Enabled = enabled == 1
		models = append(models, m)
	}
	return models, nil
}

func (s *DB) UpsertModel(modelID, name, provider string, weight, priority int, enabled bool) error {
	e := 0
	if enabled {
		e = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO model_configs (model_id, name, provider, weight, priority, enabled)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(model_id) DO UPDATE SET name=excluded.name, provider=excluded.provider, weight=excluded.weight, priority=excluded.priority, enabled=excluded.enabled
	`, modelID, name, provider, weight, priority, e)
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
