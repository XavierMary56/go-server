# PHP Yaf 内容审核对接文档

审核服务地址：`https://ai.a889.cloud`（Key 管理也在此后台）

---

## 一、建表

执行 `sql/create_moderation_logs_table.sql` 创建审核日志表，或运行迁移文件：

```php
(new Migration20260323_001())->up();
```

---

## 二、配置

在 `conf/application.ini` 对应环境节中填写：

```ini
[develop.moderation]
endpoint    = "https://ai.a889.cloud"
api_key     = "51dm_ghi789"
timeout     = 5
async       = false
webhook_url = "http://91pa.test/api.php/moderation/callback"
strictness  = "standard"

[product.moderation]
endpoint    = "https://ai.a889.cloud"
api_key     = "51dm_ghi789"
timeout     = 5
async       = false
webhook_url = "https://dx-075-api1.ympxbys.xyz/api.php/moderation/callback"
strictness  = "standard"
```

| 参数 | 说明 |
|------|------|
| endpoint | 审核服务地址，固定填 `https://ai.a889.cloud` |
| api_key | 在审核平台后台创建项目后生成的 Key |
| timeout | 请求超时秒数 |
| async | `false` 同步（直接返回结果）/ `true` 异步（回调通知） |
| webhook_url | 异步时审核服务回调你项目的地址，需公网可访问 |
| strictness | 审核严格度：`standard` / `strict` / `loose` |

```ini
[moderation]
endpoint = "https://ai.a889.cloud"
api_key  = "your_project_key"
timeout  = 5
async    = false
webhook_url = "http://your-domain.com/moderation/callback"
strictness = "standard"
```

## 三、路由注册

在 `Bootstrap.php` 中注册回调接口路由：

```php
$router->addRoute('moderation_callback', new Yaf_Route_Static('/moderation/callback'));
$router->addRoute('moderation_status',   new Yaf_Route_Static('/moderation/status'));
```

---

## 四、调用审核

将 `ContentModerationService.php` 放入 `application/library/service/`，在内容保存后调用：

```php
// 同步审核（推荐，直接拿到结果）
$result = ContentModerationService::submitForModeration(
    $post->id,       // 内容ID
    'post',          // 内容类型：post / comment / video 等
    ['content' => $post->content],
    (string) $currentUserId
);

if ($result === null) {
    // 审核服务异常，按业务决定是否放行
}

if ($result['verdict'] === 'rejected') {
    // 违规，拒绝发布
    throw new Exception('内容违规：' . $result['category']);
}

// approved = 通过，flagged = 疑似（可人工复核），均可正常发布
```

审核结果字段：

| 字段 | 值 | 说明 |
|------|----|------|
| verdict | `approved` / `flagged` / `rejected` | 审核结论 |
| category | `none` / `spam` / `abuse` / `adult` / `politics` / `fraud` / `violence` | 违规类型 |
| confidence | 0.0 ～ 1.0 | 置信度 |
| reason | 字符串 | 审核说明 |

---

## 五、异步模式（可选）

适合视频、长文等耗时较长的内容，提交后立即返回，审核完成由服务端回调通知。

```php
// 提交异步审核
$result = ContentModerationService::submitForModerationAsync(
    $video->id,
    'video',
    ['content' => $video->description],
    (string) $currentUserId
);
// $result['task_id'] 可记录日志

// 审核完成后，go-server 会 POST 到 webhook_url，
// ModerationCallbackController::callbackAction 自动处理并更新数据库状态
```

使用异步前确保：
1. `async = true` 且 `webhook_url` 填写正确
2. 回调地址公网可访问
3. 回调接口路由已注册（见第三步）

---

## 六、文件清单

```
conf/application.ini                                  配置文件
sql/create_moderation_logs_table.sql                  建表 SQL
migrations/20260323_001.php                           建表迁移
application/library/service/ContentModerationService.php   审核核心类
application/models/ModerationLog.php                  日志 Model
application/controllers/ModerationCallbackController.php   回调 + 状态查询接口
application/modules/Admin/controllers/ModerationLogsController.php  后台列表
```
