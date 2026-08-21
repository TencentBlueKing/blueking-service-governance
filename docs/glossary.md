# 词汇表（Glossary）

> 本项目涉及的核心概念、术语和缩写定义。Agent 和人类成员均以此为术语的唯一解释来源。

## Harness Engineering 核心概念

| 术语 | 英文 | 定义 |
|------|------|------|
| 驾驭工程 | Harness Engineering | 为 AI Agent 构建可靠运行环境的方法论：通过工具、上下文、约束、验证机制的系统化设计，弥补模型本身能力的不足 |
| 上下文工程 | Context Engineering | 确保 Agent 能获取准确、及时、适量上下文信息的实践，核心是知识来源管理与渐进式披露 |
| 架构约束 | Architectural Constraints | 通过刚性规则（分层、依赖方向、Linter）保证代码结构一致性的机制 |
| 熵管理 | Entropy Management | 通过文档一致性检测、技术债追踪等机制控制系统复杂度增长速度 |
| 工具能力 | Tooling | Agent 可调用的 Skill、MCP、CLI 工具清单及其接口规范与稳定性保障 |
| 执行与验证 | Execution & Verification | Agent 任务执行循环与宣称完成前的强制验证机制 |
| 局部入口 / 工作单元 AGENTS | Work-unit AGENTS.md | 非根目录下的 `AGENTS.md`，记录该子树的局部约定，优先于根 `AGENTS.md` |
| nearest 优先 | Nearest-first | 修改某路径前，应阅读该路径向上最近的 `AGENTS.md`，局部约定优先于根 |

## 架构与设计模式

| 术语 | 英文 | 定义 |
|------|------|------|
| 领域包 | Domain package | `bkms-server/pkg/<domain>/` 下按业务领域组织的代码单元，内含 `router.go`、`handler/`、`serializer/` |
| Parse, Don't Validate | Parse, Don't Validate | 在数据进入系统边界处一次性解析为强类型结构，后续代码只处理已验证数据的原则 |
| Import Cycle | 导入环 | Go 包之间相互引用形成的循环依赖；`bkms-dev-ginapi` 约定通过 router 只定义接口、上层完成构造来规避 |

## Skill 相关术语

| 术语 | 英文 | 定义 |
|------|------|------|
| Skill 安装根 | Skill Install Root（`$SKILL_ROOT`） | Agent 执行期扫描项目级 Skill 的根目录，本仓探测结果为 `.agents/skills`（`.cursor`/`.codebuddy`/`.claude`/`.codex` 均为指向 `.agents` 的软链） |
| 项目自有工具 | Project-owned tools | 目标仓自行编写并纳入 git 跟踪的 Skill/MCP，区别于 Harness 基线（治理仓统一分发） |
| graphify | graphify | 将代码/文档转为持久化知识图谱的 Skill，支撑 `docs/dev-map/` 的 query/path/explain 查询 |

## 工具与平台

| 术语 | 英文/缩写 | 定义 |
|------|----------|------|
| TAPD | Tencent Agile Product Development | 腾讯敏捷产品研发协作平台，用于需求/缺陷/迭代管理 |
| 工蜂 | Gongfeng | 腾讯内部代码托管与 Issue/MR 管理平台 |
| BCS | Blueking Container Service | 蓝鲸容器管理平台 |
| MCP | Model Context Protocol | Agent 与外部服务交互的标准化协议 |

## 项目业务术语

| 术语 | 英文/缩写 | 定义 |
|------|----------|------|
| 服务治理 | Service Governance | 面向游戏开发者、SRE 的应用全生命周期管理（托管、交付、联调、观测、策略、部署） |
| bkms-server | — | 蓝鲸服务治理服务端，Gin REST API，提供空间、环境、应用、组件、部署等领域能力 |
| bkms-cli | — | 蓝鲸服务治理命令行工具，支持应用信息查看、构建、部署、发布等操作 |

---

*持续补充中——遇到新术语时请直接在对应分类下添加。*
