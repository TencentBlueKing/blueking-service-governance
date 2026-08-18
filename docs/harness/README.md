# Harness Engineering 规范

> 本目录定义了项目的 AI Agent 运行环境规范，是 Agent 理解项目边界、工具能力和行为约束的唯一来源。

## 项目概述

- **项目名称**：蓝鲸服务治理平台（blueking-service-governance）
- **技术栈**：Go（Gin REST API、Cobra CLI）+ Vue 3 + TypeScript + Helm
- **Agent 适用场景**：bkms-server 新增/修改 REST API、bkms-cli 新增子命令、bkms-ui 页面与组件开发、数据库迁移编写、跨组件架构调整

## 规范导航

| 组件 | 文档 | 概要 |
|------|------|------|
| 上下文工程 | [context-engineering.md](context-engineering.md) | 知识来源、上下文结构、动态数据接入 |
| 架构约束 | [architectural-constraints.md](architectural-constraints.md) | 组件分层模型、依赖规则、Linter 配置 |
| 熵管理 | [entropy-management.md](entropy-management.md) | 文档一致性、技术债信号、代码评审规范 |
| 工具能力 | [tooling.md](tooling.md) | Skill/MCP/CLI 清单、接口规范、稳定性保障 |
| 执行与验证 | [execution-verification.md](execution-verification.md) | 各组件构建/测试/Lint 命令、验证机制 |

## 使用说明

1. Agent 首次接触项目时，先读本文件获取全局视图
2. 执行具体任务时，按需深入阅读对应组件文档
3. 规范更新后需同步检查关联组件的一致性（见「熵管理」）
