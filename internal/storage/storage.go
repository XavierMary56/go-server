package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB 数据库实例
type DB struct {
	db *sql.DB
}

// ProjectKey 项目密钥
type ProjectKey struct {
	ID          int64      `json:"id"`
	ProjectName string     `json:"project_name"`
	Key         string     `json:"key"`
	RateLimit   int        `json:"rate_limit"`
	Enabled     bool       `json:"enabled"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AnthropicKey Anthropic API Key
type AnthropicKey struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Key        string     `json:"key"`
	Enabled    bool       `json:"enabled"`
	Status     string     `json:"status"` // healthy | unhealthy | unknown
	UsageCount int64      `json:"usage_count"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CheckedAt  *time.Time `json:"checked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ProviderKey OpenAI / Grok API Key
type ProviderKey struct {
	ID         int64      `json:"id"`
	Provider   string     `json:"provider"` // openai | grok
	Name       string     `json:"name"`
	Key        string     `json:"key"`
	Enabled    bool       `json:"enabled"`
	Status     string     `json:"status"` // healthy | unhealthy | unknown
	UsageCount int64      `json:"usage_count"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CheckedAt  *time.Time `json:"checked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	ID       int64  `json:"id"`
	ModelID  string `json:"model_id"`
	Name     string `json:"name"`
	Provider string `json:"provider"` // anthropic | openai | grok
	Weight   int    `json:"weight"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// AdminSetting stores lightweight admin console settings.
type AdminSetting struct {
	Key       string     `json:"key"`
	Value     string     `json:"value,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// NewForTest creates a DB for use in tests. It reads TEST_DB_DSN from the
// environment and skips the test if the variable is not set.
func NewForTest(t interface {
	Helper()
	Skipf(format string, args ...interface{})
}) *DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skipf("TEST_DB_DSN not set, skipping database test")
	}
	db, err := New(dsn)
	if err != nil {
		panic("NewForTest: " + err.Error())
	}
	return db
}

// New 初始化数据库
func New(dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	s := &DB{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return s, nil
}

// migrateV2 保留空实现，MariaDB 建表时已包含所有列
func (s *DB) migrateV2() {}

func (s *DB) migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS project_keys (
			id           BIGINT AUTO_INCREMENT PRIMARY KEY,
			project_name VARCHAR(128) NOT NULL DEFAULT '',
			` + "`key`" + ` VARCHAR(256) NOT NULL UNIQUE,
			rate_limit   INT         NOT NULL DEFAULT 60,
			enabled      TINYINT(1)  NOT NULL DEFAULT 1,
			deleted_at   DATETIME    DEFAULT NULL,
			created_at   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS anthropic_keys (
			id           BIGINT AUTO_INCREMENT PRIMARY KEY,
			name         VARCHAR(128) NOT NULL,
			` + "`key`" + ` VARCHAR(512) NOT NULL UNIQUE,
			enabled      TINYINT(1)   NOT NULL DEFAULT 1,
			status       VARCHAR(32)  NOT NULL DEFAULT 'unknown',
			usage_count  BIGINT       NOT NULL DEFAULT 0,
			last_used_at DATETIME     DEFAULT NULL,
			checked_at   DATETIME     DEFAULT NULL,
			created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS provider_keys (
			id           BIGINT AUTO_INCREMENT PRIMARY KEY,
			provider     VARCHAR(32)  NOT NULL,
			name         VARCHAR(128) NOT NULL,
			` + "`key`" + ` VARCHAR(512) NOT NULL UNIQUE,
			enabled      TINYINT(1)   NOT NULL DEFAULT 1,
			status       VARCHAR(32)  NOT NULL DEFAULT 'unknown',
			usage_count  BIGINT       NOT NULL DEFAULT 0,
			last_used_at DATETIME     DEFAULT NULL,
			checked_at   DATETIME     DEFAULT NULL,
			created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS model_configs (
			id       BIGINT AUTO_INCREMENT PRIMARY KEY,
			model_id VARCHAR(128) NOT NULL UNIQUE,
			name     VARCHAR(128) NOT NULL,
			provider VARCHAR(32)  NOT NULL DEFAULT '',
			weight   INT          NOT NULL DEFAULT 50,
			priority INT          NOT NULL DEFAULT 1,
			enabled  TINYINT(1)   NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS admin_settings (
			` + "`key`" + ` VARCHAR(128) PRIMARY KEY,
			value      TEXT        NOT NULL,
			updated_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// ── Project Keys ──────────────────────────────────────────

func (s *DB) ListProjectKeys() ([]*ProjectKey, error) {
	rows, err := s.db.Query("SELECT id, project_name, `key`, rate_limit, enabled, deleted_at, created_at, updated_at FROM project_keys WHERE deleted_at IS NULL ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*ProjectKey
	for rows.Next() {
		k := &ProjectKey{}
		var enabled int
		err := rows.Scan(&k.ID, &k.ProjectName, &k.Key, &k.RateLimit, &enabled, &k.DeletedAt, &k.CreatedAt, &k.UpdatedAt)
		if err != nil {
			return nil, err
		}
		k.Enabled = enabled == 1
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *DB) GetEnabledProjectKey(key string) (*ProjectKey, error) {
	row := s.db.QueryRow("SELECT id, project_name, `key`, rate_limit, enabled, deleted_at, created_at, updated_at FROM project_keys WHERE `key`=? AND enabled=1 AND deleted_at IS NULL LIMIT 1", key)

	projectKey := &ProjectKey{}
	var enabled int
	err := row.Scan(
		&projectKey.ID,
		&projectKey.ProjectName,
		&projectKey.Key,
		&projectKey.RateLimit,
		&enabled,
		&projectKey.DeletedAt,
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

func (s *DB) AddProjectKey(projectName, key string, rateLimit int) (*ProjectKey, error) {
	now := time.Now()
	result, err := s.db.Exec(
		"INSERT INTO project_keys (project_name, `key`, rate_limit, enabled, deleted_at, created_at, updated_at) VALUES (?, ?, ?, 1, NULL, ?, ?)",
		projectName, key, rateLimit, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &ProjectKey{ID: id, ProjectName: projectName, Key: key, RateLimit: rateLimit, Enabled: true, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *DB) UpdateProjectKey(currentKey string, projectName *string, newKey *string, enabled *bool, rateLimit *int) error {
	if projectName != nil {
		_, err := s.db.Exec("UPDATE project_keys SET project_name=?, updated_at=? WHERE `key`=? AND deleted_at IS NULL", *projectName, time.Now(), currentKey)
		if err != nil {
			return err
		}
	}
	if newKey != nil {
		_, err := s.db.Exec("UPDATE project_keys SET `key`=?, updated_at=? WHERE `key`=? AND deleted_at IS NULL", *newKey, time.Now(), currentKey)
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
		_, err := s.db.Exec("UPDATE project_keys SET enabled=?, updated_at=? WHERE `key`=? AND deleted_at IS NULL", e, time.Now(), currentKey)
		if err != nil {
			return err
		}
	}
	if rateLimit != nil {
		_, err := s.db.Exec("UPDATE project_keys SET rate_limit=?, updated_at=? WHERE `key`=? AND deleted_at IS NULL", *rateLimit, time.Now(), currentKey)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DB) DeleteProjectKey(key string) error {
	_, err := s.db.Exec("UPDATE project_keys SET enabled=0, deleted_at=?, updated_at=? WHERE `key`=? AND deleted_at IS NULL", time.Now(), time.Now(), key)
	return err
}

func (s *DB) GetAdminSetting(key string) (*AdminSetting, error) {
	row := s.db.QueryRow("SELECT `key`, value, updated_at FROM admin_settings WHERE `key`=? LIMIT 1", key)

	setting := &AdminSetting{}
	err := row.Scan(&setting.Key, &setting.Value, &setting.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return setting, nil
}

func (s *DB) SetAdminSetting(key, value string) error {
	_, err := s.db.Exec(
		"INSERT INTO admin_settings (`key`, value, updated_at) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE value=VALUES(value), updated_at=VALUES(updated_at)",
		key, value, time.Now())
	return err
}

func (s *DB) GetEnabledProjectKeys() ([]string, error) {
	rows, err := s.db.Query("SELECT `key` FROM project_keys WHERE enabled=1")
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
	rows, err := s.db.Query("SELECT id, name, `key`, enabled, status, usage_count, last_used_at, checked_at, created_at FROM anthropic_keys ORDER BY created_at DESC")
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

func (s *DB) GetAnthropicKeyByID(id int64) (*AnthropicKey, error) {
	row := s.db.QueryRow("SELECT id, name, `key`, enabled, status, usage_count, last_used_at, checked_at, created_at FROM anthropic_keys WHERE id=?", id)
	k := &AnthropicKey{}
	var enabled int
	if err := row.Scan(&k.ID, &k.Name, &k.Key, &enabled, &k.Status, &k.UsageCount, &k.LastUsedAt, &k.CheckedAt, &k.CreatedAt); err != nil {
		return nil, err
	}
	k.Enabled = enabled == 1
	return k, nil
}

func (s *DB) AddAnthropicKey(name, key string) (*AnthropicKey, error) {
	now := time.Now()
	result, err := s.db.Exec(
		"INSERT INTO anthropic_keys (name, `key`, enabled, created_at) VALUES (?, ?, 1, ?)",
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

func (s *DB) UpdateAnthropicKeyName(id int64, name string) error {
	_, err := s.db.Exec(`UPDATE anthropic_keys SET name=? WHERE id=?`, name, id)
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
	rows, err := s.db.Query("SELECT id, name, `key`, enabled, status, usage_count, last_used_at, checked_at, created_at FROM anthropic_keys WHERE enabled=1 ORDER BY CASE status WHEN 'healthy' THEN 0 WHEN 'unknown' THEN 1 ELSE 2 END, usage_count ASC")
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
	rows, err := s.db.Query("SELECT id, provider, name, `key`, enabled, status, usage_count, last_used_at, checked_at, created_at FROM provider_keys WHERE provider=? ORDER BY created_at DESC", provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []*ProviderKey
	for rows.Next() {
		k := &ProviderKey{}
		var enabled int
		if err := rows.Scan(&k.ID, &k.Provider, &k.Name, &k.Key, &enabled, &k.Status, &k.UsageCount, &k.LastUsedAt, &k.CheckedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		k.Enabled = enabled == 1
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *DB) GetProviderKeyByID(id int64) (*ProviderKey, error) {
	row := s.db.QueryRow("SELECT id, provider, name, `key`, enabled, status, usage_count, last_used_at, checked_at, created_at FROM provider_keys WHERE id=?", id)
	k := &ProviderKey{}
	var enabled int
	if err := row.Scan(&k.ID, &k.Provider, &k.Name, &k.Key, &enabled, &k.Status, &k.UsageCount, &k.LastUsedAt, &k.CheckedAt, &k.CreatedAt); err != nil {
		return nil, err
	}
	k.Enabled = enabled == 1
	return k, nil
}

func (s *DB) GetEnabledProviderKeys(provider string) ([]*ProviderKey, error) {
	rows, err := s.db.Query("SELECT id, provider, name, `key`, enabled, status, usage_count, last_used_at, checked_at, created_at FROM provider_keys WHERE provider=? AND enabled=1 ORDER BY CASE status WHEN 'healthy' THEN 0 WHEN 'unknown' THEN 1 ELSE 2 END, usage_count ASC", provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []*ProviderKey
	for rows.Next() {
		k := &ProviderKey{}
		var enabled int
		if err := rows.Scan(&k.ID, &k.Provider, &k.Name, &k.Key, &enabled, &k.Status, &k.UsageCount, &k.LastUsedAt, &k.CheckedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
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
	result, err := s.db.Exec("INSERT INTO provider_keys (provider, name, `key`, enabled, created_at) VALUES (?, ?, ?, 1, ?)", provider, name, key, now)
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

func (s *DB) UpdateProviderKeyName(id int64, name string) error {
	_, err := s.db.Exec(`UPDATE provider_keys SET name=? WHERE id=?`, name, id)
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
		ON DUPLICATE KEY UPDATE name=VALUES(name), provider=VALUES(provider), weight=VALUES(weight), priority=VALUES(priority), enabled=VALUES(enabled)
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
