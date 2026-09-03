# 配置文件 Def 三表拆分与分层 Service 设计

## 背景

原有 `app_config_files` 单表承载了配置文件的逻辑身份（名称、类型）和环境实例内容，导致：

- 同一逻辑文件的多个环境实例缺少统一标识，查询和管理依赖 `appID + name` 拼接
- 无法为不同种类的配置文件（framework / 未来的 plain）附加差异化元数据
- Service 层职责混杂，既处理 CRUD 又耦合了 ConfigKind 相关的校验逻辑

## 数据模型

拆分为三张表：

```
app_config_file_defs   1 ──── N   app_config_files   1 ──── N   app_config_file_versions
（逻辑身份）                        （环境实例内容）                  （版本历史）
```

### AppConfigFileDef（新增）

| 字段          | 说明                                     |
|---------------|------------------------------------------|
| appID         | 所属应用                                 |
| name          | 配置文件名，同一应用下唯一               |
| configKind    | 配置种类（当前仅 `framework`）           |
| envConfigMode | 环境策略：是否统一配置、已挂载的环境列表 |

### AppConfigFile（改造）

新增 `defID` 字段关联 def；`name` 不再冗余存储，统一从 def 获取。内容字段抽取为 `VersionedContent` 内嵌结构，与 `AppConfigFileVersion` 共享。

### AppConfigFileVersion（改造）

同样新增 `defID`，内容字段改用 `VersionedContent` 内嵌。

## 分层 Service 架构

```
Handler Layer
  ├── app_config_file.go   (旧 handler，TODO: CLI 切换后删除)
  └── app_config_file_def.go  (新 handler)
         │
Service Layer
  ├── AppConfigFileService       ← 向后兼容门面，签名不变，内部委托 AppCfgFileDefService
  ├── AppCfgFileDefService       ← 场景层，感知 ConfigKind，选择对应 Policy 执行校验
  └── BaseAppCfgFileService      ← 底层，不感知 ConfigKind，操作三表模型的纯 CRUD
         │
Store Layer
  ├── AppConfigFileDefStore  (新增)
  ├── AppConfigFileStore      (改造，新增 defID 相关方法)
  └── AppConfigFileVersionStore
```

> **演进计划**：CLI 全部切换到新接口后，移除 `AppConfigFileService` 门面及旧 handler，
> 所有调用方直接使用 `AppCfgFileDefService`。

### ConfigKindPolicy

每个 ConfigKind 注册一个 policy，定义：

- **ValidateContent** — 内容校验（framework 校验 YAML 合法性）
- **EnvInstanceStrategy** — 环境实例产生方式（overlay / overwrite）
- **IsAlwaysMount** — 是否始终挂载到环境（framework 为 true；plain 为 false，再按 MountedEnvNames 判断）

AppCfgFileDefService 根据 def 的 ConfigKind 查表获取 policy，底层 BaseAppCfgFileService 无感。

## 兼容策略

- 旧 `AppConfigFileService` 改为门面，公开方法签名不变，内部构造 `AppCfgFileDefService` 并委托
- 旧 handler 路由保留，标注 TODO
- 新功能通过 `AppCfgFileDefService` + 新 handler 提供
- DB migration 使用 partial index（`defID.$exists`）兼容尚未回填的历史数据

## 文件结构

```
appcfg/
├── models.go              # 持久化模型：Def, AppConfigFile, Version, VersionedContent 等
├── types.go               # 枚举、常量、错误、参数类型
├── store.go               # AppConfigFileStore（改造）
├── store_def.go          # AppConfigFileDefStore（新增）
├── config_kind_policy.go  # ConfigKindPolicy + frameworkPolicy
├── service_base.go        # BaseAppCfgFileService
├── service_app_cfg_file_def.go # AppCfgFileDefService（场景层）
├── service.go             # AppConfigFileService（门面，兼容原入口）
├── handler/
│   ├── app_config_file.go # 旧 handler（TODO: 删除）
│   └── app_config_file_def.go  # 新 handler
```
