# app_config_files def 三表拆分

将配置文件的逻辑身份（名称、类型）与内容记录分离。

## 变更

### 新增集合 `app_config_file_defs`

存储配置文件的逻辑身份信息（name, configKind, envConfigMode）。

- 唯一索引 `appID_1_name_1`：同一应用下配置文件名唯一。
- 普通索引 `appID_1_configKind_1`：按 configKind 过滤查询。

### `app_config_files` 新增索引

- 唯一索引 `defID_1_envName_1`（partial）：同一 def 下每个环境最多一条记录。
- 普通索引 `defID_1`（partial）：按 def 查询所有环境实例。

两个索引均使用 `partialFilterExpression` 以兼容 defID 尚未回填的历史数据。

## 数据迁移

up 脚本通过聚合管道自动从 `app_config_files` 生成 `app_config_file_defs` 并回写 `defID`。
其中：

- `name` 从默认配置文件记录复制到 `app_config_file_defs.name`
- 默认实例的 `_id` 直接复用为对应 def 的 `_id`

> **注意**：framework 配置的 `mountDir` 当前仍由 `app_models.workload` 管理，
> 部署路径从 workload config 读取，暂不回填到 def。待 mountDir 所有权迁移至 def 后再补充。
