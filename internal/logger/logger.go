package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

var levelOrder = map[string]int{
	LevelDebug: 0,
	LevelInfo:  1,
	LevelWarn:  2,
	LevelError: 3,
}

// Logger 日志记录器
type Logger struct {
	mu     sync.Mutex
	dir    string
	level  string
	stdout bool
}

type logEntry struct {
	TS    string                 `json:"ts"`
	Level string                 `json:"level"`
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// New 创建日志实例
func New(dir, level string) *Logger {
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建日志目录失败: %v\n", err)
	}
	return &Logger{
		dir:    dir,
		level:  level,
		stdout: true,
	}
}

func (l *Logger) Debug(msg string, data map[string]interface{}) {
	l.write(LevelDebug, msg, data)
}

func (l *Logger) Info(msg string, data map[string]interface{}) {
	l.write(LevelInfo, msg, data)
}

func (l *Logger) Warn(msg string) {
	l.write(LevelWarn, msg, nil)
}

func (l *Logger) Error(msg string) {
	l.write(LevelError, msg, nil)
}

func (l *Logger) write(level, event string, data map[string]interface{}) {
	if levelOrder[level] < levelOrder[l.level] {
		return
	}

	entry := logEntry{
		TS:    time.Now().Format(time.RFC3339),
		Level: strings.ToUpper(level),
		Event: event,
		Data:  data,
	}

	line, _ := json.Marshal(entry)
	lineStr := string(line) + "\n"

	// 控制台输出
	if l.stdout {
		fmt.Print(lineStr)
	}

	// 写入文件（按天切割）
	filename := filepath.Join(l.dir, fmt.Sprintf("moderation_%s.log", time.Now().Format("2006-01-02")))
	l.mu.Lock()
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(lineStr)
		f.Close()
	}
	l.mu.Unlock()
}
