# AppModel standard 类型 · 二期收尾清单

一期已落地：`Application` 加顶层 `Language`/`Framework`、`AppTypeStandard` + `FrameworkBlank/Trpc/Taf`、
`IsAppModelType` 加 standard、创建链路（`standard.Service.Create`）、部署链路（`/standard-deploys`）。

本文档整理一期之后的收尾事项，覆盖前端 / 后端 / API / CLI / DB 层 / 冗余字段等。

## 一、DB 层变更 / 冗余字段收敛（重点）

1. **存量回填**：给存量 trpc/taf 应用回填 `framework`（trpc→`trpc`、taf→`taf`）+ 顶层 `language`
   （trpc 的 `TrpcSpec.Language` 迁到顶层）。
2. **`Application.TrpcSpec{Language}` 收敛**：一期后 standard 用顶层 `Language`，但 trpc 仍用
   `TrpcSpec.Language`，且 `AppInfoOutputObj.Language`（`serializer/app.go:373`）只读 `TrpcSpec`，
   → 需统一读顶层 Language（否则 standard 的 language 在列表接口不显示）。
3. **`Workload.TrpcConfig.Language` 冗余**：与顶层 `Language` 重复，收敛。
4. **两套语言常量统一**：`appmodel/entities.go`(go/cpp) vs `trpc/admincmd`(5 语言) 合并。
5. **`TafConfig`/`TrpcConfig` 值类型** → 后续由 framework 驱动。

## 二、后端框架分支重挂（27 处 `AppTypeTRPC/TAF` → framework）

- `apm.go` 的 YAML/XML 解析 switch
- `admin_cmd.go` 的 `app.Type` guard + `TrpcSpec.Language` switch
- `devmode`（`devmode.go` + `appspec/sections/devmode/spec.go`）
- `build/service.go` platform 构建（trpc-go only）
- `bscpcfg/metadata.go` trpc-only workloadKind
- `workspace/comp_ref.go` trpc 过滤
- `polaris/trpc_validate.go`（`WorkloadTypeTrpc`）
- 创建/更新 handler 的 TrpcSpec/TafSpec 分支

## 三、workload 插件重组织

- trpc/taf/standard 三个 plugin → framework 预设（trpc-go / trpc-cpp / taf / blank）
- `GetWorkloadPlugin` 默认 trpc → blank/standard

## 四、plain 渲染接通（配置文件）

- 部署 plugin 从 `GetEnvContent`(Deprecated) 迁到 `MountableFileProvider`/`ListPlainMountableFiles`
  （当前 plain 文件写得进库但挂不进容器）
- 补 plain 的 FileFormat（任意文本）
- framework 的 mountDir 迁到 def

## 五、API 收尾

- `UpdateAppStandardSpec`（一期未做 spec 更新接口）
- 一键构建部署 `/standard-build-deploys`
- 旧 CRUD API（appcfg 旧 framework-only 路径）清理

## 六、CLI（bkms-cli）

- `constant.go` 加 `AppTypeStandard` + language/framework
- 创建/部署/更新命令适配 standard

## 七、前端（bkms-ui）

- 应用创建表单加 standard 类型 + language/framework 选择
- 部署/实例/spec 更新页适配
- 详情页展示 language/framework

## 八、其他清理

- `AppValuesConfigStore` 死接口
- `devmode.go` 的 WorkloadType vs AppType 混用
- `AppDetailOutputObj`/`AppInfoOutputObj` 输出 language/framework

## 九、trpc/taf 收归 standard（framework 数据驱动，已定方案）

核心：把 trpc/taf 的框架差异（创建过程逐行同构，只有 3 处差异——配置文件 Format、Workload.Type、
框架配置字段）收归到 `standard`，用 `app.Framework` 驱动，达到数据驱动、少依赖 appType 判断。

已拍板 3 项决策：

1. **Workload.Type 一律收敛为 `standard`**（框架不决定部署形态，框架差异全走 FrameworkPreset）。
2. **TrpcConfig/TafConfig 统一为 `FrameworkConfig{FileName, FilePath, FileContent}`**（Language 已上提顶层 `app.Language`）。
3. **FrameworkPreset 最小化：只放 `ConfigFormat`**（治理能力清单后续再扩）。

### 第一步：框架预设 + 统一创建

1. `appmodel` 包加 `FrameworkPreset{ConfigFormat}` + `frameworkPresets` map（7 个 framework 值：
   blank=无、taf=taf、trpc-*=yaml）。
2. `Workload` 加 `FrameworkConfig` 字段（FileName/FilePath/FileContent，替代 TrpcConfig/TafConfig）。
3. `standard.Service.Create` 读 `frameworkPresets[app.Framework].ConfigFormat` 决定是否/如何建配置文件；
   `Workload.Type` 一律 `WorkloadTypeStandard`；写 `FrameworkConfig`。
4. 序列化：`AppModelSpecInput` 加框架配置输入 + `ToStandardCreateParams` 读它；handler 传递 framework config。

### 第二步：渲染/治理按 framework 分派（= 上文「二」「三」的细化）

- `standard/plugin.go` 由 no-op 改为读 framework：trpc-* → polaris patcher + config 渲染；taf → config 渲染；
  blank → no-op。
- admin/APM/devmode 由 `switch app.Type` 改为 `switch app.Framework`（后续把治理能力清单也放进预设）。

---

## 优先级建议

- **P0**：一（冗余字段收敛，尤其 `AppInfoOutputObj.Language` 读 TrpcSpec 导致 standard 语言不显示）
- **P0**：四（plain 渲染接通，否则 standard 的 plain 配置文件实际不生效）
- **P0**：九（trpc/taf 收归 standard —— framework 数据驱动的核心架构工作，第一步「框架预设+统一创建」）
- **P1**：二（框架分支重挂）+ 三（插件重组织）+ 五（API 收尾）
- **P2**：六（CLI）+ 七（前端）+ 八（清理）
