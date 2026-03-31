# PHP Yaf 内容审核对接示例

生产环境地址：`https://zyaokkmo.cc`

## 1. 配置 (conf/application.ini)

```ini
[common]
moderation.endpoint = "https://zyaokkmo.cc"
moderation.api_key  = "your_project_key"
moderation.timeout  = 5
```

## 2. 调用示例

```php
$result = ContentModerationService::moderate($content, 'post');

if ($result && $result['verdict'] === 'rejected') {
    // 内容违规，禁止发布
    exit('内容含有违规信息: ' . $result['category']);
}
```

## 3. 说明
- 本示例仅展示如何对接 `/v2/moderations` 接口。
- 不需要本地数据库表结构。
- 建议在业务代码中对审核结果进行缓存或记录日志。
