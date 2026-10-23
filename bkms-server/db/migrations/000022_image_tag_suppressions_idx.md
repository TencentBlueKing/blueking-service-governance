# 镜像 tag suppression 索引

`image_tag_suppressions` 用于记录“已确认 manifest 不存在、刷新时先不要从 `tags/list` 重新写回”的 tag，
来源是 BKMS 删除镜像，或详情同步确认 manifest 404。它不是审计日志，而是临时抑制记录：

- 当同名镜像真实重新存在时，异步复核会删除 suppression 并恢复快照；
- 当远端 `tags/list` 已经不再返回这个名字时，刷新流程会直接清理 suppression。

需要的索引很简单：

- `idx_repoKey_tag`（unique）  
  支撑 `repoKey + tag` 的幂等 upsert、定点删除与存在性判断。
- `idx_repoKey`  
  支撑按仓库实例批量列举当前全部 suppression，供 refresh 计算差集与清理。
