# PHP Yaf 客户端内容审核对接示例

本目录为内容审核服务 PHP Yaf 客户端对接的完整示例，目录结构与实际项目保持一致，便于直接参考和集成。

## 快速集成说明

请参考本节了解如何将内容审核服务集成到您的 PHP Yaf 客户端项目中。

### 主要内容
- 审核接口调用方式
- 回调处理逻辑
- 审核记录存储与查询
- 管理后台审核记录展示

详细集成步骤见下文。

## 内容审核服务集成指南

本指南详细介绍如何在 PHP Yaf 客户端项目中集成内容审核服务，包括接口调用、回调处理、审核记录存储与后台展示。

### 步骤概览
1. 引入 ContentModerationService.php
2. 配置审核服务相关参数
3. 在内容保存后调用审核接口
4. 实现审核回调接口
5. 设计审核记录表结构
6. 后台管理审核记录

### 详细说明
- 参考 application/library/service/ContentModerationService.php 代码注释
- 参考本 README.md 快速集成说明

## 配置说明

1. 在 config 或 application/configs/application.ini（或 config.php/yml/env 等）中增加如下配置：

```ini
[moderation]
endpoint = "http://moderation-api.example.com"
api_key  = "your_project_key"
timeout  = 5
async    = false
webhook_url = "http://your-domain.com/moderation/callback"
strictness = "standard"
```

或 PHP 配置数组：

```php
'moderation' => [
    'endpoint'     => 'http://moderation-api.example.com',
    'api_key'      => 'your_project_key',
    'timeout'      => 5,
    'async'        => false,
    'webhook_url'  => 'http://your-domain.com/moderation/callback',
    'strictness'   => 'standard',
],
```

2. 如果支持 .env，可添加：

```
MODERATION_ENDPOINT=http://moderation-api.example.com
MODERATION_API_KEY=your_project_key
MODERATION_TIMEOUT=5
MODERATION_ASYNC=false
MODERATION_WEBHOOK_URL=http://your-domain.com/moderation/callback
MODERATION_STRICTNESS=standard
```

3. ContentModerationService.php 会自动读取 Yaf_Registry::get('config')->moderation 或 env 变量。

4. webhook_url 需保证外部审核平台可访问，并在路由中注册该接口。

5. 迁移数据库，执行 migrations/20260323_001.php 创建 moderation_logs 审核日志表。

## conf/develop.ini 配置示例及参数说明

在 conf/develop.ini 文件中添加如下内容：

```ini
[moderation]
endpoint     = "http://moderation-api.example.com"   ; 审核服务API地址
api_key      = "your_project_key"                    ; 项目在审核平台的唯一key
timeout      = 5                                     ; 审核接口请求超时时间（秒）
async        = false                                 ; 是否异步审核（true/false）
webhook_url  = "http://your-domain.com/moderation/callback" ; 审核平台回调通知地址
strictness   = "standard"                            ; 审核严格度（standard/strict等）
```

**参数说明：**
- endpoint：内容审核服务的API基础地址，所有审核请求都发往此地址。
- api_key：你在审核平台申请的项目唯一标识，用于接口鉴权。
- timeout：调用审核API的超时时间，单位为秒，建议5~10秒。
- async：是否异步审核。true 表示提交后由回调通知审核结果，false 表示接口直接返回审核结果。
- webhook_url：审核平台回调你项目的接口地址，需保证公网可访问，通常为 http(s)://your-domain.com/moderation/callback。
- strictness：审核严格度，可选值如 standard、strict，根据业务需求选择。
