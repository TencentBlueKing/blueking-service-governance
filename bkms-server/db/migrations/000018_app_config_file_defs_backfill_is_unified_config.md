# app_config_file_defs：统一存量环境配置字段

## 背景

此前环境配置字段已按顶层 inline 设计，`000011` 等 migration 也按顶层写入了 `isUnifiedConfig`、`mountedEnvNames`，但业务代码没有真正按 inline 读写。本次把 def 模型改成 `bson:",inline"`，与已有字段对齐并真正生效。

存量数据里可能同时存在以下问题：

1. `000011_app_config_files_def_split` 在拆分 `app_config_file_defs` 时，会从默认配置文件记录生成 def：

```json
"isUnifiedConfig": { "$ifNull": ["$isUnifiedConfig", false] }
```

   拆分前的 `app_config_files` 存量数据通常没有 `isUnifiedConfig` 字段，因此会被统一刷成
   `false`。这会把“只有默认文件、并没有任何环境实例”的 def 错误标记成“按环境独立配置”。

2. 历史代码曾把环境配置信息写入嵌套字段 `envConfigMode`，而不是顶层字段。

3. 个别 framework 环境实例文件的 `defID` 可能失效，但 `baseAppConfigFileID` 仍指向正确的默认文件。

## 统一规则

- 存在 `envName` 非空的环境实例：`isUnifiedConfig = false`
- 不存在任何环境实例：启发式回填为 `true`

> 说明：
> 这条 migration 用于统一存量数据的 `isUnifiedConfig` 取值，按“当前是否存在环境实例”做一次性回填。
> 运行时语义仍以服务逻辑为准；若用户在新接口中进入按环境模式后又删除了最后一个环境实例，
> 当前实现不会自动把 `isUnifiedConfig` 改回 `true`。

## 执行顺序

先执行完本 migration，再让读取顶层字段的新二进制接流量。新代码使用 `bson:",inline"`，读不到嵌套的 `envConfigMode`；若新二进制先接流量，这些 def 会被当成按环境独立配置。

## 迁移动作

迁移按以下顺序执行：

1. 修复 orphan 的环境实例文件：
   - 若环境实例文件的 `defID` 指向不到任何 def
   - 则先通过 `baseAppConfigFileID` 找到对应的默认文件
   - 优先取默认文件上的 `defID` 作为候选值；若默认文件本身没有 `defID`，再退化使用 `baseAppConfigFileID`
   - 当候选值能对应到现存 def，且目标 `(candidateDefID, envName)` 还没有其他实例占用时，将该文件的 `defID` 回写为候选值

   同一批里如果有两条失效环境文件的 `envName` 和 `baseAppConfigFileID` 都相同，第二条写入会撞上 `defID + envName` 唯一索引，migration 中断并标记为 dirty。已改过的 `defID` 不会自动回滚。处理方式：按下面「验证」第 2 节查出仍失效的文件，确认没有两条要改挂到同一个 `(defID, envName)` 后，再重新执行。这种重复数据很少见。

2. 对全部 def，以 `app_config_file_defs` 为主表按 def `_id` 回查 `app_config_files`：
   - 若存在任意一条 `defID = 当前 def._id` 且 `envName` 非空的记录，则顶层 `isUnifiedConfig` 刷为 `false`
   - 否则顶层 `isUnifiedConfig` 刷为 `true`

3. 对仍存在 `envConfigMode` 的 def：
   - 将 `envConfigMode.mountedEnvNames` 写到顶层 `mountedEnvNames`；嵌套里没有该字段时保留顶层原值
   - 删除嵌套字段 `envConfigMode`
   - 不使用 `envConfigMode.isUnifiedConfig`。旧代码把顶层文档解码成零值后再 Update，会把凭空的 `false` 写进嵌套字段

4. 放宽挂载目录唯一约束：
   - 删除 `appID_1_mountDir_1`（同一应用下 `mountDir` 单独唯一，禁止同目录多文件）
   - 创建 `appID_1_mountDir_1_name_1`（unique，partial：`mountDir` 存在且非空）
   - 最终容器挂载路径是 `filepath.Join(mountDir, name)`，同目录不同文件名合法；`appID + name` 唯一索引保留

`isUnifiedConfig` 只由第 2 步按环境实例是否存在决定。嵌套字段只提供 `mountedEnvNames`。

## 已执行过旧版本时

`schema_migrations` 已经是 18 的库不会再次执行修改后的 `up`（含本次并入的索引变更）。这些库需要先 `migrate force 17`，再 `migrate up`。数据回填不能靠 down 恢复旧值。

若某库已经执行过已删除的 `000019`，游标会停在 19 而仓库里没有对应文件，需要 `migrate force 18` 把游标拉回。索引若已建好可不动；若还没有，再 `force 17` 后 `up`。

## down

数据回填部分为 no-op，只回滚第 4 步的索引：删掉 `appID_1_mountDir_1_name_1`，重建 `appID_1_mountDir_1`。

`isUnifiedConfig` / `envConfigMode` 回滚到迁移前的旧值没有业务意义，且无法安全区分：

- 哪些 `false` 是修复前误刷出来的错误值
- 哪些 `false` 是本就正确的“按环境独立配置”值

若 down 时已经写入「同目录多文件」，重建旧索引会失败，需先清掉冲突数据。

## 验证

```js
// 1. 不应再有 envConfigMode 嵌套字段
db.app_config_file_defs.find(
  { envConfigMode: { $exists: true } },
  { appID: 1, name: 1, configKind: 1, envConfigMode: 1 }
)

// 2. 不应再有 env file 的 defID 指向不到任何 def
db.app_config_files.aggregate([
  {
    $match: {
      envName: { $exists: true, $nin: ["", null] }
    }
  },
  {
    $lookup: {
      from: "app_config_file_defs",
      localField: "defID",
      foreignField: "_id",
      as: "defs"
    }
  },
  {
    $match: {
      defs: { $size: 0 }
    }
  }
])

// 2.1 不应再有 orphan env file 因目标 (defID, envName) 冲突而跳过修复
db.app_config_files.aggregate([
  {
    $match: {
      envName: { $exists: true, $nin: ["", null] },
      baseAppConfigFileID: { $exists: true, $type: "objectId" }
    }
  },
  {
    $lookup: {
      from: "app_config_file_defs",
      localField: "defID",
      foreignField: "_id",
      as: "defs"
    }
  },
  {
    $match: {
      defs: { $size: 0 }
    }
  }
])

// 3. 全部 def 的顶层 isUnifiedConfig 应与 defID 关联的环境实例数量一致
db.app_config_file_defs.aggregate([
  {
    $lookup: {
      from: "app_config_files",
      localField: "_id",
      foreignField: "defID",
      as: "files"
    }
  },
  {
    $project: {
      name: 1,
      isUnifiedConfig: 1,
      envInstanceCount: {
        $size: {
          $filter: {
            input: "$files",
            as: "f",
            cond: { $gt: [{ $strLenCP: { $ifNull: ["$$f.envName", ""] } }, 0] }
          }
        }
      }
    }
  }
])
```

检查结果中应满足：

- 不再有文档残留 `envConfigMode`
- 不再有环境实例文件的 `defID` 指向不到任何 def
- `envInstanceCount > 0` 的 def，其 `isUnifiedConfig` 为 `false`
- `envInstanceCount == 0` 的 def，其 `isUnifiedConfig` 为 `true`
