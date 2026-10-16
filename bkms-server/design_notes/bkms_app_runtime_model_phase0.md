# BKMSApp 运行时模型：阶段 0 基线与冻结决策

状态：已冻结，作为阶段 1 及后续实现的唯一决策来源。父文档：[bkms_app_runtime_model.md](./bkms_app_runtime_model.md)。

检索命令与分类方法沿用父文档第 8 节；本文件记录分类结论，而不是要求仓库中的 `trpc` / `taf` 文本消失。

## 1. 冻结决策

| 决策 | 冻结值 | 理由与安全边界 |
| --- | --- | --- |
| BKMSApp 持久化字面值 | `bkmsApp` | JSON/BSON/API 第一份新字段起使用该值，不再讨论驼峰或连字符变体 |
| 框架配置字段 | `frameworkConfig`（对象）、`frameworkConfigVersion`（int，初始 1） | 只存框架扩展；`fileName` / `filePath` 首期放 map；`fileContent` 不进 map |
| TAF 支持语言 | 目录声明 **仅 cpp** | 现有模型/API/Bruno 均无 TAF language 字段。存量未扫描前 **不得** 批量写 `language=cpp`。兼容读取可将 framework 投影为 `taf`，language 留空并记 `incomplete_language` |
| 技术栈可变性 | 普通更新 **不可变** | `PUT /trpc-spec` 今日会改 `Workload.TrpcConfig.Language` 但不改 `Application.TrpcSpec`。兼容窗口内：旧入口若提交与当前生效 language **不同** 的值，返回明确的“不支持切换技术栈”错误（该校验落在后续写路径收拢批次，不在阶段 1 读模型里改写行为） |
| 兼容发布 | 桥接版本 → 客户端同时识别新旧 → 数据切换 → 删旧入口 | 窗口内详情/列表 **继续输出存量 `type=trpc\|taf`**；只增量返回 `language` / `framework`。窗口内 **不开放 blank 创建**，不把 type 切成 `bkmsApp` |
| 客户端兼容窗口 | 阶段 4 切换 type 响应前，UI/CLI 必须同时识别 `trpc` / `taf` / `bkmsApp` | 未完成客户端迁移前，服务端创建仍写旧 type。阶段 1 不改创建入口 oneof |
| Helm / Agones | 不改模型 | 不自动填 language/framework；若存量意外带了这些字段，扫描记 `unexpected_stack_fields`，不覆盖 HelmSpec |
| 空 Workload.Type | **不**再视为 tRPC | 兼容读取不套用 plugin 的空值兜底。仅旧插件注册表在渲染路径保持原行为，直到阶段 3 改分派 |

阶段 1 垂直切片：框架目录、Application 新字段、旧数据规范化与一致性测试。不启用 `type=bkmsApp` 创建，不改渲染输出。

## 2. 技术栈与能力矩阵（当前行为，非目标宣传）

`supported` 表示 **今天代码真实开放的路径**。blank 仅目录合法，阶段 1 不创建。

| 能力 | trpc+go | trpc+cpp | taf（无 language） | blank+go/cpp | helm | agones |
| --- | --- | --- | --- | --- | --- | --- |
| 管理方式 AppModel | 是 | 是 | 是 | 目标 | 否 | 否 |
| `appModelDeploy` 部署/状态/实例 | 是 | 是 | 是 | 阶段 3 | Helm 独立链路 | Agones 独立链路 |
| `plainConfigFiles` | 是 | 是 | 是 | 目标 | n/a | n/a |
| `frameworkConfig` 文件 | YAML | YAML | XML | 无伪框架文件 | n/a | n/a |
| `devMode` | 是 | 是 | 是（按 TAF 路径） | 否 | 否 | 否 |
| `platformBuild` | 是 | 否 | 否 | 否 | 否 | 否 |
| `adminCommands` | go/cpp（另有 java/node/python **读配置** 分支，非创建语言） | 同左 | 是 | 否 | 否 | 否 |
| `apmServiceName` | YAML 解析 | YAML 解析 | XML 解析 | 否 | 否 | 否 |
| `polarisFrameworkPatch` | 是 | 是 | 否 | 否 | 否 | 否 |
| BSCP workload 元信息 | 仅 `type=trpc` | 同左 | **未接** | 否 | 否 | 否 |
| 组件引用查询 | 仅 `type=trpc` | 同左 | **未查** | 否 | 否 | 否 |
| 集群 addon `requiredForAppTypes` | `trpc`,`taf` | 同左 | 同左 | 阶段 3E | 部分 addon 含 helm/agones | 同左 |

合法组合（阶段 1 目录）：`trpc+go`、`trpc+cpp`、`taf+cpp`、`blank+go`、`blank+cpp`。合法不等于功能全开：`blank+go` 不自动获得 platformBuild。

## 3. 旧分支消费清单（分类）

下列为阶段 0 全库检索后的归类。完整路径以父文档第 2.1 节与仓库检索为准；阶段 1 只要求 **没有未经解释的类型消费点**。

### 3.1 管理方式（AppModel vs Helm/Agones）

`IsAppModelType` 与 `type in {trpc,taf}` 用于部署、实例、拓扑、WebConsole、环境变量、镜像、日志、异步轮询、workspace 过滤。阶段 1：`IsAppModelType` **同时识别** `trpc` / `taf` / `bkmsApp`。UI/CLI 的 `APP_MODEL_APP_TYPES`、旧部署路由仍留阶段 4。

代表位置：`pkg/core/app/app.go`、`pkg/deploy/**`、`pkg/workload/instance/**`、`pkg/workload/topology`、`pkg/server/taskqtask/appmodeldeploypoll`、`bkms-ui/src/composables/app-type.ts`、`bkms-cli/pkg/constant`。

### 3.2 能力选择（应用 type/language 决定功能开关）

| 消费点 | 当前规则 | 阶段 1 | 后续 |
| --- | --- | --- | --- |
| platformBuild | `type==trpc` 且 `TrpcSpec.Language==go` | 不改执行 | 阶段 2 Mod |
| devmode | 非 TAF 走 tRPC 路径 | 不改执行；记为隐式兜底 | 阶段 2 去掉 else |
| BSCP metadata | 仅 trpc | 保持 | 阶段 3D |
| `comp_ref.go` | 仅列出 trpc | **保持行为**，单独立项，不在本迁移里“顺便修” | 评估是否补 taf/bkmsApp |
| cluster addon YAML | required/optional 含 trpc/taf | 保持 | 阶段 3E 拆管理方式与能力 |
| UI 导航/构建表单 | 按 type/language 硬编码 | 保持 | 阶段 4 改读 RuntimeModSet |

### 3.3 协议实现（保留内部 trpc/taf 包）

`pkg/workload/appmodelcore/trpc`、`taf` 的创建/更新、plugin 渲染、admincmd、Polaris 校验、APM YAML/XML 解析。阶段 1 只增加类型化 `DecodeConfig`，**不改渲染挂载名/路径**（仍以 AppModel 的 fileName/filePath 为准）。

### 3.4 兼容 / 历史数据

- 双 language：`Application.TrpcSpec` vs `Workload.TrpcConfig`；更新路径只写后者。
- 列表 language 来自 Application；详情嵌套 `trpcSpec.language` 来自 Workload。
- `Workload.Type` 空 → plugin 默认 tRPC。
- `fileContent` 仍带 bson 标签；`pkg/core/render/migrate/draft.go` 处理 taf 残留。
- dbfactory `TrpcApplication` 常不写 `Application.TrpcSpec`。
- 部署记录 / 异步任务 payload 存 `AppType: "trpc"`。
- Swagger、`bkms-ui/src/@types/v1`、Bruno `trpc-spec` / `taf-spec` / `trpc-deploys`。

`libs/bkms-adapter` 无 AppType 耦合。`bkms-server/configs` 与 `db` 迁移 JSON 无 trpc/taf 应用类型字面值（addon 资产 YAML 除外）。

## 4. 已知不一致（保持 vs 修复）

| 问题 | 处理 |
| --- | --- |
| 两处 language 冲突 | 阶段 1 Normalize **报错**，不静默覆盖、不以 go 补齐 |
| 仅一处 language 有效 | 使用有效值并记录来源 |
| TAF 无 language | 读取不猜测；迁移待存量核对 |
| 列表 vs 详情 language 来源不同 | 阶段 1 列表/空间列表增量 `framework`，language 优先 `Application.Language` 再 `TrpcSpec`；详情嵌套 `trpcSpec` **保持原 Workload 来源**，避免改渲染/旧客户端 |
| 空 Workload.Type → tRPC | 新规范化不套用；旧 plugin 暂留 |
| 组件引用只查 trpc | **保持**，单独修复 |
| BSCP 只接 trpc | **保持**，阶段 3D |
| dbfactory 缺 TrpcSpec | 测试夹具问题；不改默认以免大面积测试漂移 |
| admincmd java/node/python | 非产品创建语言；扫描遇未知 language 记 `unknown_language` |
| fileContent 残留 | 阶段 5 清理前只扫描，不删字段 |

## 5. 列表 / 筛选方案（阶段 1 约定）

对外详情/列表在兼容窗口：

- `type`：仍为库存值（`trpc` / `taf` / `helm` / `agones`）。**不**输出 `bkmsApp`，直到阶段 4 切换门槛满足。
- `language`：已有列表字段。优先 `Application.Language`，否则 `TrpcSpec.Language`，否则 `""`。
- `framework`：新增。优先 `Application.Framework`，否则由旧 type `trpc`/`taf` 投影，Helm/Agones 为空。

筛选语义（store 层先落地，HTTP 查询参数阶段 4 再暴露，以免未迁移客户端误用）：

- 旧 `type=trpc` ≡ `{type: "trpc"}` **或** `{type: "bkmsApp", framework: "trpc"}`，在数据库过滤，不在内存截断分页。
- `type=taf` 同理。
- `type=helm|agones|bkmsApp` 精确匹配。
- 新字段 `framework` / `language` 过滤器只匹配已回填的新字段；未回填旧记录不会命中，避免把空 language 的 TAF 误当成 cpp。

## 6. API 与渲染基线样本

阶段 1 **不得改变** 下列请求/渲染语义。新字段只追加读取。

### 6.1 创建 tRPC（Bruno `tests/apis/apps/trpc-app/create-trpc-app.bru`）

- `POST /workspaces/:id/apps`，`type=trpc`，`appModelSpec.trpcSpec.language` 为 `go` 或 `cpp`。
- 响应 `data.type` 仍为 `trpc`。
- 更新：`PUT /apps/:id/trpc-spec`，body 含 `appModelSpec.trpcSpec.language`（今日会写入 Workload）。

### 6.2 创建 TAF（Bruno `tests/apis/apps/taf-app/create-taf-app.bru`）

- `type=taf`，`tafSpec` 仅 `fileName` / `filePath` / `fileContent`，无 language。
- 响应 `data.type` 仍为 `taf`。

### 6.3 渲染

- tRPC plugin 用 `Workload.TrpcConfig.FileName` / `FilePath` 作为挂载名与目录；内容来自 MountableFileProvider；**不用** def 的 name/mountDir。
- 空 `Workload.Type` 时 `GetWorkloadPlugin` 回退 tRPC。阶段 1 不删除该回退。
- Helm/Agones 不走 AppModel 渲染。

回归对照组合：`trpc+go`、`trpc+cpp`、`taf`、一组 Helm、一组 Agones。阶段 1 用单元测试锁规范化等价性；清单字节级对比属于阶段 3A。

## 7. 脱敏存量扫描

### 7.1 工具

命令：`bkms-server scan_app_runtime_model --srvCfg <file>`（只读）。

输出：计数（按 type、issue code）+ 样本（`appID`、旧 type、两处 language、workload type、是否双框架配置、是否有 fileContent **布尔**）。不输出配置正文、凭据、镜像 pull secret。

冲突种类（扫描与 Normalize 共用）：

| code | 含义 |
| --- | --- |
| `language_conflict` | Application / TrpcSpec / TrpcConfig 语言非空且不一致 |
| `type_framework_conflict` | App type、Application.Framework、非空 Workload.Type 指向不同框架 |
| `unknown_language` | 非 go/cpp（含 java/node/python 存量） |
| `unknown_framework` / `unknown_app_type` | 目录外取值 |
| `invalid_combination` | 如 taf+go |
| `incomplete_language` | 无有效 language；TAF 为 advisory，tRPC/blank/新 bkmsApp 为 blocking |
| `dual_framework_config` | trpcConfig 与 tafConfig 同时非空 |
| `framework_config_conflict` | 新旧 fileName/filePath 不一致 |
| `unknown_config_version` / `missing_config_version` | map 已写但 version 非法或缺失 |
| `unexpected_stack_fields` | Helm/Agones 带了 language/framework |
| `orphan_application` | AppModel 类型应用缺少 AppModel |

### 7.2 本工作区执行情况

本实现环境 **没有** 生产 Mongo 副本，因此没有线上脱敏报表。阶段 0 交付以：

1. 代码与测试可构造的冲突种类（见上表）；
2. 只读扫描命令；
3. 用测试库插入合成记录验证计数与样本不含 fileContent。

上线前必须在只读副本执行该命令，把 TAF 是否全为 C++、以及 java/node/python 出现次数补进本文件第 7.3 节。

### 7.3 生产扫描占位（待填）

| 项 | 结果 |
| --- | --- |
| 扫描时间 / 集群 | _待填_ |
| type=trpc / taf / helm / agones 数量 | _待填_ |
| language 冲突条数 | _待填_ |
| TAF 是否全部可确认为 cpp | _待填_ |
| 未知 language 样本（仅 appID） | _待填_ |
| 双框架配置 / 空 Workload.Type / fileContent 残留 | _待填_ |

## 8. 后续模块边界

| 模块 | 阶段 1 负责 | 不在阶段 1 |
| --- | --- | --- |
| `pkg/core/appruntime` | 值类型、目录、Inspect/Normalize、严格 map 解码、列表 type 展开 | RuntimeModSet resolver |
| `pkg/core/app` | 新字段、IsAppModelType、store 兼容读与 type 展开 | 创建改写为 bkmsApp |
| `appmodel` | frameworkConfig 字段、旧 TrpcConfig/TafConfig 作为兼容 DTO | 删除旧字段、改 plugin 分派 |
| serializer / workspace 列表 | 增量 framework；language 优先新字段 | 改 type 响应、新创建/更新 API |
| trpc/taf | DecodeConfig / 旧配置转 map | 写路径双写、渲染改读 map |
| 扫描命令 | 只读报告 | apply/回填（阶段 5） |

## 9. 阶段 1 验收对照

- 旧 BSON 往返不丢 `trpcSpec` / `trpcConfig` / `tafConfig`。
- 旧输入与新输入 Normalize 到同一 `Stack`（冲突除外）。
- 冲突返回 issue，不选某一侧覆盖。
- 新增字段不参与阶段 1 渲染；plugin 仍读旧字段。
- 目录内每个合法/非法组合有测试；`blank` 与空字符串不同。
