package admin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// trainingRecord 训练数据记录（精简结构，只保留训练所需字段）
type trainingRecord struct {
	Content    string  `json:"content"`              // 审核内容
	Verdict    string  `json:"verdict"`              // 审核结果: approved / rejected
	Category   string  `json:"category,omitempty"`   // 分类: adult, fraud, politics...
	Confidence float64 `json:"confidence,omitempty"` // 置信度 0-1
	Reason     string  `json:"reason,omitempty"`     // 拒绝原因
	Type       string  `json:"type,omitempty"`       // 内容类型
	Model      string  `json:"model,omitempty"`      // 使用的模型
	Project    string  `json:"project,omitempty"`     // 来源项目
	Timestamp  string  `json:"timestamp"`            // 时间戳
}

// handleExportTrainingData 处理 GET /v1/admin/export/training-data
// 流式导出审核日志为训练数据（JSONL 格式）
// 参数：
//   - project: 项目ID（可选，为空则导出所有项目）
//   - start: 开始日期 2006-01-02（可选，默认30天前）
//   - end: 结束日期 2006-01-02（可选，默认今天）
//   - min_confidence: 最低置信度 0-1（可选，默认0，即全部）
//   - format: 导出格式 jsonl/csv（可选，默认jsonl）
func (ah *AdminHandler) handleExportTrainingData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ah.jsonError(w, http.StatusMethodNotAllowed, "仅支持 GET")
		return
	}

	q := r.URL.Query()
	projectID := q.Get("project")
	startStr := q.Get("start")
	endStr := q.Get("end")
	minConfStr := q.Get("min_confidence")
	format := q.Get("format")
	if format == "" {
		format = "jsonl"
	}

	// 解析日期范围
	var startTime, endTime time.Time
	if startStr != "" {
		t, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			ah.jsonError(w, http.StatusBadRequest, "invalid start date")
			return
		}
		startTime = t
	} else {
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	}

	if endStr != "" {
		t, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			ah.jsonError(w, http.StatusBadRequest, "invalid end date")
			return
		}
		endTime = t.Add(24 * time.Hour)
	} else {
		endTime = time.Now().Add(24 * time.Hour)
	}

	// 解析最低置信度
	var minConfidence float64
	if minConfStr != "" {
		v, err := strconv.ParseFloat(minConfStr, 64)
		if err != nil {
			ah.jsonError(w, http.StatusBadRequest, "invalid min_confidence")
			return
		}
		minConfidence = v
	}

	// 确定要扫描的项目目录
	var projectDirs []string
	if projectID != "" {
		projectDirs = []string{projectID}
	} else {
		projectDirs = ah.collectAllProjectIDs()
	}

	// 设置响应头 - 流式下载
	filename := fmt.Sprintf("training_data_%s.%s", time.Now().Format("20060102_150405"), format)
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	// CSV 头（写入 UTF-8 BOM，确保 Excel 正确识别中文编码）
	if format == "csv" {
		w.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM
		fmt.Fprintf(w, "\"content\",\"verdict\",\"category\",\"confidence\",\"reason\",\"type\",\"model\",\"project\",\"timestamp\"\n")
	}

	var exported int

	// 流式扫描每个项目目录
	for _, pid := range projectDirs {
		dir := filepath.Join(ah.cfg.AuditLogDir, pid)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		// 按文件名倒序排列（最新日期在前）
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() > entries[j].Name()
		})

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
				continue
			}

			// 通过文件名快速过滤日期范围（audit_2026-04-03.log）
			datePart := strings.TrimPrefix(entry.Name(), "audit_")
			datePart = strings.TrimSuffix(datePart, ".log")
			if fileDate, err := time.Parse("2006-01-02", datePart); err == nil {
				if fileDate.Before(startTime) || fileDate.After(endTime) {
					continue
				}
			}

			filePath := filepath.Join(dir, entry.Name())
			exported += ah.streamTrainingRecords(w, flusher, canFlush, filePath, pid, startTime, endTime, minConfidence, format)
		}
	}

	// JSONL 尾部写入统计注释行（不影响解析）
	if format == "jsonl" && exported == 0 {
		fmt.Fprintf(w, "{\"_info\":\"no matching records found\"}\n")
	}

	if canFlush {
		flusher.Flush()
	}
}

// streamTrainingRecords 从单个日志文件中流式提取训练数据
func (ah *AdminHandler) streamTrainingRecords(
	w http.ResponseWriter,
	flusher http.Flusher,
	canFlush bool,
	filePath string,
	projectID string,
	startTime, endTime time.Time,
	minConfidence float64,
	format string,
) int {
	f, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer f.Close()

	var count int
	scanner := bufio.NewScanner(f)
	// 增大 buffer 以处理包含长内容的日志行
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// 快速预筛：必须是 moderation_request
		if !strings.Contains(string(line), "moderation_request") {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		// 只导出 moderation_request 事件
		if event["event_type"] != "moderation_request" {
			continue
		}

		// 时间范围过滤
		if ts, ok := event["timestamp"].(string); ok {
			t, err := time.Parse(time.RFC3339Nano, ts)
			if err == nil && (t.Before(startTime) || t.After(endTime)) {
				continue
			}
		}

		reqBody, _ := event["request_body"].(map[string]interface{})
		metadata, _ := event["metadata"].(map[string]interface{})

		if reqBody == nil || metadata == nil {
			continue
		}

		content, _ := reqBody["content"].(string)
		verdict, _ := metadata["verdict"].(string)
		if content == "" || verdict == "" {
			continue
		}

		// 置信度过滤
		confidence, _ := metadata["confidence"].(float64)
		if minConfidence > 0 && confidence < minConfidence {
			continue
		}

		category, _ := metadata["category"].(string)
		reason, _ := metadata["reason"].(string)
		contentType, _ := reqBody["type"].(string)
		model, _ := metadata["model_used"].(string)
		timestamp, _ := event["timestamp"].(string)

		if format == "csv" {
			fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
				csvQuote(content),
				csvQuote(verdict),
				csvQuote(category),
				csvQuote(fmt.Sprintf("%.4f", confidence)),
				csvQuote(reason),
				csvQuote(contentType),
				csvQuote(model),
				csvQuote(projectID),
				csvQuote(timestamp),
			)
		} else {
			rec := trainingRecord{
				Content:    content,
				Verdict:    verdict,
				Category:   category,
				Confidence: confidence,
				Reason:     reason,
				Type:       contentType,
				Model:      model,
				Project:    projectID,
				Timestamp:  timestamp,
			}
			data, err := json.Marshal(rec)
			if err != nil {
				continue
			}
			w.Write(data)
			w.Write([]byte("\n"))
		}

		count++

		// 每 100 条刷新一次，避免内存堆积
		if canFlush && count%100 == 0 {
			flusher.Flush()
		}
	}

	return count
}

// csvQuote 强制给所有 CSV 字段加双引号并转义内部双引号
// 确保 Excel 在中文 Windows 下正确识别 UTF-8 编码的中文内容
func csvQuote(s string) string {
	return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
}
