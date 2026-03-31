# 数据库迁移说明

## 迁移方式

迁移逻辑写在 `internal/storage/storage.go` 的 `migrate()` 函数中，**服务启动时自动执行**，无需手动跑 SQL。

所有迁移语句必须幂等（重复执行不报错）。

## 新增迁移的做法

在 `storage.go` 的 `migrate()` 函数末尾追加迁移逻辑，格式参考已有示例：

```go
// 迁移：描述（幂等）
var count int
s.db.QueryRow(`SELECT COUNT(*) FROM information_schema.COLUMNS WHERE ...`).Scan(&count)
if count > 0 {
    s.db.Exec(`ALTER TABLE ...`)
}
```

## 迁移历史

| 序号 | 说明 | 日期 |
|------|------|------|
| 001 | project_keys 表字段 project_id 重命名为 project_name | 2026-04-01 |
