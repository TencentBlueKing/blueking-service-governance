# 集群插件说明

## 概念说明

集群插件（cluster_addons）与空间/应用组件（components）是完全不同的两个概念：

- **集群插件（cluster_addons）**：部署在 K8S 集群特定命名空间中，为集群层面提供增强能力，如 CLB 集成、北极星服务注册中心、监控数据上报等
- **空间/应用组件（components）**：BKMS 特有概念，通过 patch 机制将自定义模板内容写入平台产物中

## 功能定位

集群插件功能为用户提供了在 BKMS 平台上快速部署和管理各类插件的能力。本质上是对 Helm 能力的封装抽象，通过调用 Helm SDK 在用户集群中部署和管理对应的 Helm Release。

## 数据结构

默认提供的集群插件定义存储于 `bkms-server/pkg/core/env/clusteraddon/assets/addons`

### MongoDB 集合：cluster_addon_defs

存储集群插件定义的信息，为插件体系的基础配置层。

| 字段名 | 类型 | 索引 | 说明 |
|--------|------|------|------|
| `name` | string | ✓ 唯一 | 插件唯一标识符，用于系统内部引用 |
| `displayName` | string | | 展示名称，用于 UI 显示 |
| `description` | string | | 插件描述，说明其功能和用途 |
| `chartInfo.chartName` | string | | Helm Chart 名称（Helm 仓库地址在全局配置中） |
| `chartInfo.defaultChartVersion` | string | | 默认使用的 Chart 版本 |
| `chartInfo.defaultNamespace` | string | | 默认安装命名空间（若未指定，使用 `bcs-system`） |
| `chartInfo.exampleValues` | string | | 安装示例参数（YAML 格式字符串，包含注释说明） |
| `requiredForAppTypes` | []string | | 应装插件的应用类型列表（e.g., `["trpc"]`） |
| `optionalForAppTypes` | []string | | 可选安装该插件的应用类型列表（e.g., `["taf"]`） |

注意，在 DB 中存储的实际上只是插件的定义，用户获取插件列表时，后台实际会根据 DB 中的定义列表，分别在预置的 Helm 仓库中获取对应 Chart 的列表（用于拼装可选的版本号等信息），然后在用户集群中获取是否有对应安装的 Helm Release 信息（用于拼装当前版本、部署参数、部署状态等信息）

Q: 为什么直接在集群中获取 Helm Release 的信息，而不在 DB 中维护对应的安装记录？
A：主要为了兼容用户通过其他方式部署的插件（如 BCS 组件库、 Helm 等）。
