---
name: Go Unit Test Expansion
description: 针对当前 Go 仓库扩展单元测试，优先覆盖未测试分支、边界条件与错误路径
tags: testing, go, unit-tests
---

# Go 单元测试扩展规则

适用于当前 Go 仓库。扩展测试时，默认遵循 `CLAUDE.md` 中已经存在的命令与目录约定。

## 一、先确认仓库测试方式
仅使用仓库已明确存在的测试命令：

- `go test ./...`
- `go test ./internal/service`
- `go test ./internal/service -run Rules`
- `go test ./internal/service -run Bulk`
- `go test ./internal/service -run TestServiceBehavior`

不要虚构或默认使用以下命令，除非仓库后续明确新增并写入 `CLAUDE.md`：

- `make test`
- `npm test`
- `pytest`
- `golangci-lint`
- `go test -cover...`

## 二、补测试的优先顺序
1. 先读现有测试和被测代码，遵循现有命名与断言风格。
2. 优先补当前改动直接影响的包，而不是一上来跑全仓。
3. 优先覆盖：
   - 错误路径
   - 边界值
   - 空输入 / 缺省值
   - 状态转换
   - 副作用与回归场景
4. 只有在局部验证通过后，再根据需要扩大到 `go test ./...`。

## 三、Go 仓库约束
- 测试文件优先与被测包放在同目录。
- 优先使用表驱动测试。
- 子测试命名应直接表达场景。
- 只写当前需求需要的测试，不为假设性未来需求扩展。
- 如果仓库内已有同类测试模式，优先复用，而不是自创结构。

## 四、执行与汇报
进行测试扩展时，优先给出：
1. 将修改哪些测试文件
2. 将运行哪些已存在的 `go test` 命令
3. 这些命令分别验证什么

输出应贴合当前仓库真实工作流，而不是通用跨语言模板。
