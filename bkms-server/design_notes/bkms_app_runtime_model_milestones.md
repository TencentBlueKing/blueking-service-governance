# BKMSApp 运行时模型：里程碑

以 [总体方案](./bkms_app_runtime_model.md) 为指导原则，按"一个里程碑一个 PR"拆分。各项设计细节、兼容约束与验证门禁以总体方案为准，本文只列每个 PR 的范围、行为变化与退出条件。

相关文档：[阶段 0 冻结决策](./bkms_app_runtime_model_phase0.md)、[能力集合方案](./bkms_app_runtime_model_phase1.md)。

## 拆分规则

- 一个里程碑只提一个 PR。
- PR 内先新增、后切换；纯目录移动单独成里程碑，不与逻辑改动混合。
- 每个里程碑结束时系统可独立上线；兼容窗口规则见总体方案 §5.3、§6.2。

## 总览

| 里程碑 | 目标 | 总体方案阶段 | 原规划步骤 | 状态 |
| --- | --- | --- | --- | --- |
| M0 | 新模型字段、框架目录、兼容读取、扫描工具 | 0、1 | — | ✅ #225 |
| M1 | 能力集合新路径（只新增，不接入） | 2 | 1 | ✅ |
| M2 | 能力执行端切换到能力集合 | 2、3D | 1 | 🚧 |
| M3 | 能力 pkg 划分（纯移动） | — | 1 | ⬜ |
| M4 | 运行链路识别 bkmsApp：渲染分派、部署预检、集群依赖等 | 3A、3C、3E | 2 | ⬜ |
| M5 | 创建与修改流程统一：编排合并、新旧字段双写、blank 后端闭环 | 3B、3F | 2 | ⬜ |
| M6 | 新 API 上线，旧 API 改薄适配层 | 4（服务端） | 2 | ⬜ |
| M7 | 前端接入，切换响应 type | 4（客户端） | 3 | ⬜ |
| M8 | 存量回填，切换主读写 | 5 | 4 | ⬜ |
| M9 | 下线旧接口，删除旧模型与兼容代码 | 6 | 5 | ⬜ |

CLI 的双识别与升级由用户跟进，需在 M7 切换 type 前发布。

---

## M1 能力集合新路径

**范围：** `runtimemod` 基础包（Spec 与查表工厂）、8 个能力声明、`Set` 与 `Of(app)`、`Public` / `Matrix`、`CapabilityStack`；taf / blank 完整 golden、`Set` 字段反射对齐、import 约束测试。

**行为变化：** 无。生产代码不调用新路径。

**退出条件：** 测试覆盖能力目录全部合法组合；`make lint`、`make test` 通过。

---

## M2 能力执行端切换

**范围：** 8 个能力的消费点改读 `mods.Of(app)`，删除对应的类型分支。

| 能力 | 切换要点 |
| --- | --- |
| devMode | 脚本集与路径只来自 props；删除"非 TAF 即 tRPC"兜底 |
| platformBuild | 可用性与 Dockerfile 语言来自 props |
| adminCommand | 协议与配置键布局来自 props。|
| apmServiceName | 解析器来自 props |
| polarisServiceCheck | 判断依据由 `Workload.Type` 改为 Application |
| polarisFrameworkPatch | 渲染 patch 链按能力判断；trpc 专用 |
| bscpWorkloadMeta | 是否自动填充按能力判断；工作负载类型取实际渲染结果，不再写死 GameDeployment |
| frameworkFile | 内容校验按 props.format 选择解析器；Helm 仍按 YAML |

**行为变化：**

- TAF 且 `Workload.Type` 为空：由 tRPC 脚本改为 TAF 脚本（修复）。
- TAF 框架文件不再被 YAML 校验误拦（修复）。
- blank 开启 devMode 返回能力错误。
- platformBuild 不支持时返回统一能力错误。
- `Workload.Type` 为空的 tRPC 应用开始收到 Polaris 校验警告。
- Helm 应用不能再通过 API 创建 plain 文件（UI 本就不暴露）。plain 与一键构建部署按基座处理，改判 `IsAppModelType`，不走能力集合。

**上线前：** 只读副本执行 `scan_app_runtime_model`，确认受影响记录数与未知 language。存量若有目录外语言，先补入目录（只做存量兼容，不开放创建），再上线。

> ⚠️ **注意：** 为存量补入目录的语言属于临时兼容项，必须在代码中以 BRIDGE-ONLY 标注，并登记到 M8 的迁移后检查项。若不清理，这些语言会长期留在目录与能力矩阵中，被误当作产品支持的语言。

**退出条件：** 能力判断场景下不再读取 `AppType` / `Workload.Type` / `TrpcSpec.Language`；每个 supported 组合有执行器覆盖测试。

---

## M3 能力 pkg 划分

**范围：** 按总体方案 §4.5 与能力集合方案 §6 移动目录：能力进 `pkg/capability/*`，框架协议实现进 `pkg/framework/*`，`Set` 的组装独立成薄包。

**行为变化：** 无，零逻辑 diff。

**退出条件：** `runtimemod` 与组装包不依赖领域包；`make lint`、`make test` 通过。

---

## M4 运行链路识别 bkmsApp

**目标：** 部署、渲染等运行链路改为按管理方式（`IsAppModelType`）或 framework 判断，不再比较 `type == "trpc"` 这类字面值。只改判断逻辑，不改数据。

**为什么需要：** 今天应用 type 只有 trpc / taf，字面值比较都能命中；blank 只能是 `bkmsApp`，M7 后所有应用也都是 `bkmsApp`，这些比较会全部落空。

| 事项 | 现状 | 改为 |
| --- | --- | --- |
| 渲染插件选择 | 按 `Workload.Type` 选插件，空值默认 trpc（`workload/builder.go`、`plugin/plugin.go`） | 按 `Application.Framework` 选；blank 对应 standard 插件；去掉空值兜底 |
| 部署预检 | 只认 trpc / taf，bkmsApp 落入 default 报错（`deploy/pre_checker.go`） | 按 `IsAppModelType` |
| 部署接口类型校验 | 严格比较 expectedAppType（`appmodel_deploy_steps.go`） | 按 framework 核对 |
| 容器名内置变量 | 只给 trpc / taf 注入（`envvars/listers.go`） | 按 `IsAppModelType` |
| 集群 addon 依赖 | 比较 type 字面值，YAML 只写 trpc / taf（`clusteraddon/query.go`） | 按管理方式与工作负载形态表达；同步内置 YAML 与库内记录 |
| 其余字面值比较 | 组件引用查询、监控指标类型归一等 | 同上 |

框架配置文件的挂载信息仍读 Workload，改读 def 在 M8（见总体方案 §3.4）。

**行为变化：** 无。

**退出条件：** trpc / taf 新旧渲染清单逐项一致（含 `trpc-init`、`/taf-config-template` 等资源名）；不再有对 trpc / taf 字面值的运行时判断；Helm / Agones 无回归。

---

## M5 创建与修改流程统一

**目标：** 合并 trpc / taf 两套创建与修改流程，写入时同时写新字段与旧投影。

| 事项 | 现状 | 改为 |
| --- | --- | --- |
| 创建编排 | 按类型分派到 `trpc.go`、`taf.go` 两套几乎相同的流程（解析默认值、建框架 def、初始化 AppModel、写 AppSpec） | 合并为一个编排服务，差异（配置格式、旧投影）由框架适配器提供，并读取能力声明 |
| 新字段写入 | 生产代码从未写过 `Application.Language/Framework` | 与旧字段同时写 |
| 框架配置文件挂载信息 | 只写 `Workload.FileName/FilePath`；改名不同步 def | 同时写框架 def 的 `Name` / `MountDir` 与 Workload（方案见总体方案 §3.4） |
| 技术栈不可变 | `trpc-spec` 更新会改 `TrpcConfig.Language` | 提交不同 language 时报"不支持切换技术栈" |
| 跨集合写入 | Application、配置文件、AppModel、AppSpec 分别写，中途失败留下半成品 | MongoDB 事务保护 |
| blank 后端闭环 | 无 | 端到端测试：已有镜像 → plain 配置 → 渲染 → 部署 → 状态 → 下架；创建入口仍关闭 |
| 旧路由调用量 | 无统计 | 埋点，作为 M9 下线依据 |

**行为变化：** 技术栈切换被拒绝；其余对外行为不变。

**退出条件：** 新旧字段写入一致；blank 后端端到端通过；创建失败不留半成品应用。

---

## M6 新 API 上线

**范围：** 总体方案 §5.1 的全部新接口；旧接口改为调用同一领域服务的薄适配层并强制核对 framework；详情 / 列表增量返回 `framework` / `language`；合并 `trpc-build-deploys` / `taf-build-deploys` 为一键构建部署接口；Swagger 与 Bruno。

**行为变化：** 只新增接口；旧接口业务效果不变，响应 `type` 仍为 `trpc|taf`。

**退出条件：** 新旧入口业务等价；Bruno 全量通过。

---

## M7 前端接入

**范围：** UI 改读能力集合与能力目录，配置、部署、构建部署改用新 API，`type` 三值识别；门槛满足后服务端输出 `type=bkmsApp` 并开放 blank 创建。

**门槛：** CLI 新版本已发布；M5 埋点显示旧客户端在观察期内无调用。

**退出条件：** UI 不再硬编码技术栈能力；三种框架产品流程联调通过。

---

## M8 存量回填

**范围：** 按总体方案 §6 执行回填与主读写切换，交付以下 mongosh 脚本并附执行说明：

| 脚本 | 作用 |
| --- | --- |
| scan | 只读统计，填写阶段 0 文档 §7.3 |
| backup | 导出待改字段与记录级清单 |
| backfill | 幂等、按原值条件更新，可分批执行；含框架 def 的 `Name` / `MountDir` 回填 |
| verify | 核对目标数与完成数、冲突清单 |
| restore | 按备份条件回滚，不覆盖迁移后的用户编辑 |

**框架 def 回填（总体方案 §3.4）：** 先执行 `repair_trpc_taf_framework_defs` 清理多余的框架 def；再逐应用比对 Workload 与 def 的文件名、挂载目录，并检查与 plain 文件的挂载路径冲突；冲突列入清单人工处理，不自动覆盖。核对一致后，渲染改读 def。

**行为变化：** 新字段成为主读写来源；旧投影继续写一个窗口。

**迁移后检查：**

- [ ] 复核 M2 为存量兼容补入目录的语言：确认对应应用已迁移或已处理，再决定从目录移除，或经产品确认后转为正式支持。
- [ ] `verify` 脚本结果中无目录外 language。
- [ ] 每个 AppModel 应用只有一条框架 def，且 `Join(MountDir, Name)` 与回填前的挂载路径一致。

**退出条件：** 冲突为零；历史部署与文件版本回放正确；恢复演练通过；迁移后检查项全部完成。

---

## M9 下线与清理

**范围：** 总体方案阶段 6：删除旧路由、旧字段（含 `Workload.TrpcConfig/TafConfig`）、旧类型常量、BRIDGE-ONLY 代码与扫描命令；清理测试工厂、Bruno、Swagger、UI 类型；提供库内旧字段清理的 mongosh 脚本。

**门槛：** 旧路由调用量在完整发布周期内归零；CLI 旧版本已淘汰。

**退出条件：** 活跃数据无旧字段；三种框架与 Helm / Agones 完整门禁通过。
