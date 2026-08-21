# 基于数据库的镜像本地缓存设计方案

## 1. 解决的问题

在应用部署或配置过程中，如果每次都向远程镜像仓库（如 BKRepo 或外部 Registry）实时拉取镜像标签（Tag）及详情（Digest、Size、BuildTime），会面临以下问题：

- 远程接口响应耗时较长，导致前端页面加载缓慢。
- 强依赖外部网络和仓库服务，容易受到限流或网络波动影响导致请求失败。

解决方案：在本地数据库（MongoDB）中建立镜像快照缓存，将镜像列表数据本地化，并通过异步刷新与详情同步机制来同步上游仓库的数据。

## 2. 核心数据模型 (MongoDB)

### 2.1 镜像快照记录 (`ImageSnapshot`)

存储单个镜像标签的具体信息：

- `RepoKey`: 仓库实例的唯一标识（基于仓库名和凭据生成）。
- `Tag`: 镜像标签名（如 `latest`, `v1.0.0`）。
- `Digest`, `Size`, `BuiltAt`: 镜像的摘要、大小和构建时间（由异步详情同步任务填充）。

### 2.2 仓库快照状态 (`RepoSnapshotStatus`)

记录整个仓库的缓存同步状态，用于并发控制和前端展示：

- `RepoKey`, `RepoName`: 仓库标识与名称。
- `RefreshStatus`: 当前刷新状态（`idle` 空闲 / `refreshing` 标签刷新中 / `detail_syncing` 详情同步中）。
- `LastRefreshedAt`, `LastDetailSyncedAt`: 最后成功刷新和详情同步的时间。
- `LastError`: 最后的失败错误信息。

## 3. 核心工作流设计

整个缓存机制分为三个主要阶段：查询触发、标签刷新、详情同步。

### 3.1 查询与按需触发 (Read & Trigger)

- 接口查询：用户请求镜像列表时，直接从 MongoDB 的 `ImageSnapshot` 集合中分页查询。
- 初始触发：如果查询发现本地无该仓库的快照记录（total = 0），则异步触发一次初始刷新（`triggerInitialRefresh`），避免阻塞当前用户请求。
- 手动刷新：用户也可以通过 API 主动触发刷新（`RefreshSnapshots`）。

### 3.2 标签刷新 (Refresh)

该阶段主要负责同步远程仓库的 Tag 列表，暂不拉取镜像详情。

1. 并发控制：利用 MongoDB 的原子操作（`TrySetRefreshing`）尝试将状态置为 `refreshing`，确保同一仓库同一时间只有一个刷新任务在运行。
2. 全量拉取：调用 Registry API 获取该仓库下的所有 Tag 列表。
3. 本地对比与更新：
   - 将获取到的 Tag 批量 Upsert 到 MongoDB（此时详情字段为空）。
   - 删除本地存在但远程已消失的 Tag 记录。
4. 状态流转：更新状态为 `idle`，记录 `LastRefreshedAt`。
5. 派发详情同步任务：将详情同步任务（`taskq.imageDetailSync`）投递到 asynq 队列，交由后台 Worker 异步处理。

### 3.3 详情同步 (Detail Sync)

该阶段由后台 Worker 消费 asynq 任务执行，负责拉取镜像的 Digest、Size 等元数据。

1. 并发控制：利用 MongoDB 的原子操作（与标签刷新阶段类似）尝试将状态置为 `detail_syncing`，确保同一仓库同一时间只有一个详情同步任务在运行。
2. 增量获取：从 MongoDB 查询出需要同步详情的 Tag（如详情字段为空的 Tag，或 `latest` 等可变 Tag）。
3. 并发控制拉取：
   - 采用分批处理（如每批 10 个）和信号量并发控制（如最大并发 3）调用 Registry API 获取 Tag 详情。
   - 避免瞬间高并发请求压垮远程镜像仓库。
4. 数据回写：将获取到的详情更新回 MongoDB 的 `ImageSnapshot` 记录中。
5. 状态流转：更新状态为 `idle`，记录 `LastDetailSyncedAt`。
