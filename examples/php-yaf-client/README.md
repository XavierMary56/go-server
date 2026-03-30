# PHP Yaf 内容审核对接文档

审核服务地址：`https://ai.a889.cloud`（Key 管理也在此后台）

---

## 一、说明

当前示例同时支持：
- **V2（推荐）**：`/v2/moderations`、`/v2/moderations/async`、`/v2/tasks/{id}`
- **V1（兼容）**：`/v1/moderate`、`/v1/moderate/async`、`/v1/task/{id}`

新项目建议优先接入 **V2**。如果已有旧项目在使用 V1，可继续保留。

---

## 二、配置

在 `conf/application.ini` 对应环境节中填写：

```ini
[develop.moderation]
endpoint    = "https://ai.a889.cloud"
api_key     = "your_project_key"
api_version = "v2"
timeout     = 5
async       = false
webhook_url = "http://91pa.test/api.php/moderation/callback"
strictness  = "standard"
```

| 参数 | 说明 |
|------|------|
| endpoint | 审核服务地址，固定填 `https://ai.a889.cloud` |
| api_key | 在审核平台后台创建项目后生成的 Key |
| api_version | `v2`（推荐）或 `v1`（兼容旧版） |
| timeout | 请求超时秒数 |
| async | `false` 同步（直接返回结果）/ `true` 异步（回调通知） |
| webhook_url | 仅异步模式需要；当前默认同步接入可不配置 |
| strictness | 审核严格度：`standard` / `strict` / `loose` |

---

## 三、路由注册

只需要一个示例路由即可（用于验证对接是否通）：

```php
$router->addRoute('moderation_demo', new Yaf_Route_Static('/moderation/demo'));
```

---

## 四、调用审核

将 `ContentModerationService.php` 放入 `application/library/service/`，在内容保存后调用：

```php
// 默认按 api_version 配置选择 V2 或 V1
$result = ContentModerationService::submitForModeration(
    $post->id,
    'post',
    ['content' => $post->content],
    (string) $currentUserId
);

if ($result === null) {
    // 审核服务异常，按业务决定是否放行
}

if ($result['verdict'] === 'rejected') {
    throw new Exception('内容违规：' . $result['category']);
}
```

### V1 与 V2 返回差异
- V1 直接返回：`verdict / category / confidence / reason / model_used / latency_ms`
- V2 原始返回包装在 `data.result`
- 示例服务类已经做了统一兼容，业务层仍可直接读取：
  - `verdict`
  - `category`
  - `confidence`
  - `reason`
  - `model_used`
  - `latency_ms`

---

## 五、异步模式

```php
$result = ContentModerationService::submitForModerationAsync(
    $video->id,
    'video',
    ['content' => $video->description],
    (string) $currentUserId
);

if ($result !== null) {
    $taskId = $result['task_id'];
}
```

任务查询同样会自动兼容 V1 / V2：

```php
$task = ContentModerationService::queryTask($taskId);
```

---

## 六、文件清单

```
conf/application.ini                                       配置文件
application/library/service/ContentModerationService.php    审核核心类（兼容 V1 / V2）
application/controllers/ModerationController.php            demo 接口（用于验证对接）
```
