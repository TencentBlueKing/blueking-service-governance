# 删除集群 Addon 安装任务集合

`000028` 为北极星组件的批量异步安装新建了 `cluster_addon_install_tasks`（`clusteraddon/installtask`）。
方案调整为由用户逐个集群安装，复用集群插件的同步安装接口
（`POST /workspaces/{ws}/clusters/{clusterID}/cluster-addons/{addonName}`），并发互斥由安装锁保证，
不再需要任务记录，模型与存储已删除，见 `design_notes/polaris_access_point.md` 5.1。

该集合从未有过写入方，直接删除集合及其索引。`000028` 已合入主干，部分环境已经执行，因此不修改它，而是新增本迁移。

- up：`drop` 集合 `cluster_addon_install_tasks`。
- down：按 `000028` 的定义重建该集合的三个索引，使回退到 `000028` 时其 down 仍能正常删除这些索引。
