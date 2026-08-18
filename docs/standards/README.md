# 技术规范

> Agent 实现需求时的开发行为准则。根据需求涉及的端按需加载对应规范。

## 必选规范（横切）

| 分类 | 规范 | 文档 | 加载 |
|------|------|------|------|
| 安全 | 蓝鲸代码安全三大红线 | [security-bk-redlines.md](security-bk-redlines.md) | 按「加载预算」读**相关红线节**（非每次全文） |
| 质量 | 代码评审规范 | [quality-code-review.md](quality-code-review.md) | **按需**：仅 Code Review / 质量评分任务（见加载预算）；日常改代码不预加载 |

> 安全为横切红线；质量长文 **opt-in（按需）**，避免默认灌入。文件仍部署到本目录供按需 Read。

## 当前项目选用的规范

| 分类 | 规范 | 文档 | 技术栈 | 项目事实（命中根） |
|------|------|------|--------|-------------------|
| 前端 | Vue 3 + TypeScript + Vite | [frontend-vue3.md](frontend-vue3.md) | vue, typescript, vite, pinia | 命中根：[`bkms-ui`](../../bkms-ui/package.json) |
| 接口 | Gin + Swagger/OpenAPI | [api-swagger.md](api-swagger.md) | gin, swagger, openapi, rest | 命中根：[`bkms-server`](../../bkms-server/go.mod) |
| 后端 | Go + Gin REST | [backend-gin.md](backend-gin.md) | go, gin, rest | 命中根：[`bkms-server`](../../bkms-server/go.mod) |

## Agent 加载步骤（强制）

1. 解析本文件「当前项目选用的规范」表，得到与本任务相关的规范**文件路径**。
2. 查「加载预算」：确定本任务要读哪些**章节**（不是整份文件）。
3. 用「章节快速索引」定位后，**按节 Read**（不要凭记忆复述；**禁止**默认全文灌入）。
4. 实现过程中若偏离规范，先说明冲突再改代码或请用户裁断。
5. IDE Rules（`standards-*.mdc` / `standards-*.md`）仅督促按节加载；完整条款以 `docs/standards/` 为准。禁止为业务规范逐文件生成 Always Rule。

## 加载预算

| 任务类型 | 应 Read | 默认不预加载 |
|---------|---------|--------------|
| 文案/样式/单行无逻辑改动 | 无，或前端「陷阱」相关小节 | 安全全文、质量评分、后端/接口全文 |
| 改前端组件/状态/路由/请求（`bkms-ui/`） | 前端规范中对应章节（见索引） | 前端全文、质量评分量表、其它端全文 |
| 改接口契约/鉴权/联调（`bkms-server/pkg/<domain>/router.go` 等） | 安全相关红线节 + 接口规范相关节 | 前端全文、质量全文 |
| 改后端业务逻辑（`bkms-server/`） | 后端规范相关节 + 安全相关红线节 | 前端全文、质量评分量表 |
| 新建敏感接口/鉴权/加密逻辑 | 安全规范对应红线整节 + 检查清单 | 无关前端/质量长文 |
| Code Review / 质量评分 | 质量入口决策表 + 相关**分册**（`./quality-code-review/`） | 日常改代码、质量评分量表（非评分任务）、前端最佳实践全书 |

## Agent 加载策略

| 需求类型 | 应加载的规范 |
|---------|------------|
| 任何涉及业务代码的需求 | 按「加载预算」读已选用安全规范的**相关节**；非每次全文 |
| 涉及 `bkms-ui/` 页面或组件 | 按节加载 `frontend-vue3.md` 对应章节 |
| 涉及 `bkms-server/` 新增 REST 接口 | 按节加载 `api-swagger.md` + `backend-gin.md` 相关节 |

## 规范约束力

- 标注"禁止"/"必须"的条目：**强制**遵守，违反需明确说明原因
- 标注"推荐"/"优先"的条目：**优先**遵守，有合理理由可偏离
- 常见场景参考：**参考**实现，可根据具体情况调整

## 章节快速索引

### frontend-vue3.md

| 章节 | 锚点 |
|------|------|
| 本次必读决策表 | frontend-vue3.md#本次必读决策表 |
| 一、技术栈要求 | frontend-vue3.md#一技术栈要求 |
| 二、项目结构 | frontend-vue3.md#二项目结构 |
| 三、编码规范 | frontend-vue3.md#三编码规范 |
| 四、状态管理规范（Pinia） | frontend-vue3.md#四状态管理规范Pinia |
| 五、网络请求规范 | frontend-vue3.md#五网络请求规范 |
| 六、三层代码架构 | frontend-vue3.md#六三层代码架构 |
| 七、Vue 3 最佳实践 | frontend-vue3.md#七Vue3最佳实践 |
| 八、UI 组件使用规范 | frontend-vue3.md#八UI组件使用规范 |
| 九、质量保证 | frontend-vue3.md#九质量保证 |
| 十、常见陷阱与避坑 | frontend-vue3.md#十常见陷阱与避坑 |
| 规范落地优先级 | frontend-vue3.md#规范落地优先级 |

### api-swagger.md

| 章节 | 锚点 |
|------|------|
| 本次必读决策表 | api-swagger.md#本次必读决策表 |
| 一、架构概述 | api-swagger.md#一架构概述 |
| 二、生成物与目录 | api-swagger.md#二生成物与目录 |
| 三、注解与 Handler 规范 | api-swagger.md#三注解与Handler规范 |
| 四、数据类型与 JSON 契约 | api-swagger.md#四数据类型与JSON契约 |
| 五、生成命令 | api-swagger.md#五生成命令 |
| 六、前后端契约同步 | api-swagger.md#六前后端契约同步 |
| 七、安全与红线（交叉引用） | api-swagger.md#七安全与红线交叉引用 |
| 八、常见陷阱 | api-swagger.md#八常见陷阱 |
| 九、质量保证 | api-swagger.md#九质量保证 |
| 规范落地优先级 | api-swagger.md#规范落地优先级 |

### backend-gin.md

| 章节 | 锚点 |
|------|------|
| 本次必读决策表 | backend-gin.md#本次必读决策表 |
| 一、技术栈要求 | backend-gin.md#一技术栈要求 |
| 二、项目结构 | backend-gin.md#二项目结构 |
| 三、编码规范 | backend-gin.md#三编码规范 |
| 四、错误处理与日志 | backend-gin.md#四错误处理与日志 |
| 五、配置、超时与安全 | backend-gin.md#五配置超时与安全 |
| 六、测试 | backend-gin.md#六测试 |
| 七、构建与质量门禁 | backend-gin.md#七构建与质量门禁 |
| 八、常见陷阱 | backend-gin.md#八常见陷阱 |
| 规范落地优先级 | backend-gin.md#规范落地优先级 |
