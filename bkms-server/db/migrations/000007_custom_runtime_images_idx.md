# 工作空间自定义运行时镜像索引

工作空间级自定义镜像（builder / runner）新增独立集合 `custom_runtime_images`，与平台官方镜像集合 `runtime_images` 分离，避免为官方镜像补 `workspaceID` 维度、改既有唯一索引并给存量数据回填。

- custom_runtime_images 新建索引：
  - workspaceID_1_type_1_name_1: CustomRuntimeImage StoreMongo：同一工作空间 + 镜像类型下，一个镜像仓库名最多一条记录；写入为幂等 upsert，唯一冲突视为成功。同一镜像同时用作 builder 与 runner 时落两条记录，由 `type` 区分。

本迁移不涉及 `runtime_images` 的任何索引变更。
