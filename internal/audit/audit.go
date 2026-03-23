package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditLogger 审计日志记录器（按项目分别存储）
type AuditLogger struct {
	mu       sync.Mutex
	baseDir  string      // 基础日志目录
	enabled  bool
	eventCh  chan *AuditEvent
	stopCh   chan struct{}
	logDirs  map[string]string // 项目ID -> 日志目录的映射
	logDirMu sync.RWMutex
}

// AuditEvent 审计事件
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`   // auth, api_call, error, etc.
	ProjectID   string                 `json:"project_id"`
	APIKey      string                 `json:"api_key"`      // 隐藏的密钥
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	StatusCode  int                    `json:"status_code"`
	LatencyMs   int64                  `json:"latency_ms"`
	ErrorMsg    string                 `json:"error_msg,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	RequestBody map[string]interface{} `json:"request_body,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// New 创建审计日志记录器
func New(baseDir string, enabled bool) *AuditLogger {
	al := &AuditLogger{
		baseDir:  baseDir,
		enabled:  enabled,
		eventCh:  make(chan *AuditEvent, 1000),
		stopCh:   make(chan struct{}),
		logDirs:  make(map[string]string),
	}

	// 创建基础日志目录
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建审计日志基础目录失败: %v\n", err)
	}

	// 后台日志记录 goroutine
	if enabled {
		go al.processEvents()
	}

	return al
}

// getProjectLogDir 获取或创建项目日志目录
func (al *AuditLogger) getProjectLogDir(projectID string) string {
	al.logDirMu.RLock()
	dir, exists := al.logDirs[projectID]
	al.logDirMu.RUnlock()

	if exists {
		return dir
	}

	// 创建项目日志目录
	sanitizedProjectID := sanitizeProjectID(projectID)
	projectDir := filepath.Join(al.baseDir, sanitizedProjectID)

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建项目日志目录失败: %v\n", err)
		return al.baseDir  // 回退到基础目录
	}

	al.logDirMu.Lock()
	al.logDirs[projectID] = projectDir
	al.logDirMu.Unlock()

	return projectDir
}

// LogEvent 记录审计事件
func (al *AuditLogger) LogEvent(event *AuditEvent) {
	if !al.enabled {
		return
	}

	select {
	case al.eventCh <- event:
	case <-al.stopCh:
	default:
		fmt.Fprintf(os.Stderr, "审计日志队列满，事件已丢弃\n")
	}
}

// LogAuthAttempt 记录认证尝试
func (al *AuditLogger) LogAuthAttempt(projectID string, apiKey string, success bool, ipAddress string) {
	event := &AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "auth_attempt",
		ProjectID:  projectID,
		APIKey:     maskKey(apiKey),
		IPAddress:  ipAddress,
		StatusCode: 200,
		Metadata: map[string]interface{}{
			"success": success,
		},
	}
	al.LogEvent(event)
}

// LogAPICall 记录 API 调用
func (al *AuditLogger) LogAPICall(projectID string, apiKey string, method string, path string,
	statusCode int, latencyMs int64, ipAddress string, errorMsg string) {

	event := &AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "api_call",
		ProjectID:  projectID,
		APIKey:     maskKey(apiKey),
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		LatencyMs:  latencyMs,
		ErrorMsg:   errorMsg,
		IPAddress:  ipAddress,
	}
	al.LogEvent(event)
}

// LogRateLimitExceeded 记录速率限制触发
func (al *AuditLogger) LogRateLimitExceeded(projectID string, apiKey string, ipAddress string) {
	event := &AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "rate_limit_exceeded",
		ProjectID:  projectID,
		APIKey:     maskKey(apiKey),
		IPAddress:  ipAddress,
		StatusCode: 429,
		ErrorMsg:   "请求过于频繁",
	}
	al.LogEvent(event)
}

// LogConfigChange 记录配置变更
func (al *AuditLogger) LogConfigChange(projectID string, changeType string, details map[string]interface{}) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "config_change",
		ProjectID: projectID,
		Metadata: map[string]interface{}{
			"change_type": changeType,
			"details":     details,
		},
	}
	al.LogEvent(event)
}

// processEvents 后台处理事件
func (al *AuditLogger) processEvents() {
	for {
		select {
		case event := <-al.eventCh:
			al.writeEvent(event)
		case <-al.stopCh:
			return
		}
	}
}

// writeEvent 写入单个事件到项目对应的文件
func (al *AuditLogger) writeEvent(event *AuditEvent) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// 获取项目日志目录
	projectLogDir := al.getProjectLogDir(event.ProjectID)

	// 按日期创建文件
	filename := filepath.Join(projectLogDir, fmt.Sprintf("audit_%s.log", time.Now().Format("2006-01-02")))

	data, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化审计事件失败: %v\n", err)
		return
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开审计日志文件失败: %v\n", err)
		return
	}
	defer f.Close()

	f.WriteString(string(data) + "\n")
}

// Close 关闭审计日志记录器
func (al *AuditLogger) Close() {
	close(al.stopCh)
	time.Sleep(100 * time.Millisecond)
}

// maskKey 隐藏 API 密钥
func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// QueryEvents 查询审计日志（按项目和时间范围）
func QueryEvents(baseLogDir string, projectID string, startTime time.Time, endTime time.Time) ([]AuditEvent, error) {
	var events []AuditEvent

	// 获取项目日志目录
	sanitizedProjectID := sanitizeProjectID(projectID)
	projectLogDir := filepath.Join(baseLogDir, sanitizedProjectID)

	// 遍历日期范围内的所有日志文件
	for d := startTime.Truncate(24 * time.Hour); !d.After(endTime.Truncate(24 * time.Hour)); d = d.Add(24 * time.Hour) {
		filename := filepath.Join(projectLogDir, fmt.Sprintf("audit_%s.log", d.Format("2006-01-02")))

		data, err := os.ReadFile(filename)
		if err != nil {
			continue // 文件不存在则跳过
		}

		for _, line := range bytes.Split(data, []byte("\n")) {
			if len(line) == 0 {
				continue
			}

			var event AuditEvent
			if err := json.Unmarshal(line, &event); err != nil {
				continue
			}

			// 过滤时间范围
			if event.Timestamp.Before(startTime) || event.Timestamp.After(endTime) {
				continue
			}

			events = append(events, event)
		}
	}

	return events, nil
}

// ListProjects 列出所有有日志的项目
func ListProjects(baseLogDir string) ([]string, error) {
	var projects []string

	entries, err := os.ReadDir(baseLogDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			projects = append(projects, entry.Name())
		}
	}

	return projects, nil
}

// GetProjectStats 获取项目统计信息
func GetProjectStats(baseLogDir string, projectID string) (map[string]interface{}, error) {
	sanitizedProjectID := sanitizeProjectID(projectID)
	projectLogDir := filepath.Join(baseLogDir, sanitizedProjectID)

	// 获取项目日志大小
	var totalSize int64
	filepath.Walk(projectLogDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	// 统计各类事件数
	stats := map[string]int{
		"api_call":             0,
		"auth_attempt":         0,
		"rate_limit_exceeded":  0,
		"config_change":        0,
	}

	entries, err := os.ReadDir(projectLogDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(projectLogDir, entry.Name()))
		if err != nil {
			continue
		}

		for _, line := range bytes.Split(data, []byte("\n")) {
			if len(line) == 0 {
				continue
			}

			var event AuditEvent
			if err := json.Unmarshal(line, &event); err != nil {
				continue
			}

			if count, exists := stats[event.EventType]; exists {
				stats[event.EventType] = count + 1
			}
		}
	}

	return map[string]interface{}{
		"project_id":           projectID,
		"total_size_bytes":     totalSize,
		"total_size_mb":        fmt.Sprintf("%.2f", float64(totalSize)/1024/1024),
		"event_counts":         stats,
	}, nil
}

// sanitizeProjectID 清理项目ID（确保可用作文件夹名）
func sanitizeProjectID(projectID string) string {
	// 只允许字母、数字、下划线
	result := ""
	for _, r := range projectID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' {
			result += string(r)
		}
	}
	if result == "" {
		result = "unknown"
	}
	return result
}
