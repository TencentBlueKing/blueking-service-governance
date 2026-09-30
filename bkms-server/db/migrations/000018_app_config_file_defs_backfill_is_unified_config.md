# app_config_file_defs：统一存量环境配置字段

## 背景

本次变更将 `AppConfigFileDef` 的环境配置字段改为顶层平铺存储：

- `isUnifiedConfig`
- `mountedEnvNames`

但存量数据里可能同时存在以下问题：

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

## 迁移动作

迁移按以下顺序执行：

1. 修复 orphan 的环境实例文件：
   - 若环境实例文件的 `defID` 指向不到任何 def
   - 则先通过 `baseAppConfigFileID` 找到对应的默认文件
   - 优先取默认文件上的 `defID` 作为候选值；若默认文件本身没有 `defID`，再退化使用 `baseAppConfigFileID`
   - 当候选值能对应到现存 def，且目标 `(candidateDefID, envName)` 还没有其他实例占用时，将该文件的 `defID` 回写为候选值

2. 对**没有** `envConfigMode` 的 def，以 `app_config_file_defs` 为主表按 def `_id` 回查 `app_config_files`：
   - 若存在任意一条 `defID = 当前 def._id` 且 `envName` 非空的记录，则顶层 `isUnifiedConfig` 刷为 `false`
   - 否则顶层 `isUnifiedConfig` 刷为 `true`

3. 对仍存在 `envConfigMode` 的 def：
   - 将 `envConfigMode.isUnifiedConfig` 覆盖到顶层 `isUnifiedConfig`
   - 将 `envConfigMode.mountedEnvNames` 覆盖到顶层 `mountedEnvNames`
   - 删除嵌套字段 `envConfigMode`

其中第 3 步用于统一历史 mixed-shape 文档，并以旧代码实际写入的 nested 值作为存量真值；
第 2 步只处理没有 nested 字段的文档，避免把第 3 步要保留的历史值提前覆盖掉。

## down

`down` 为 no-op。

原因是该迁移属于存量数据统一，回滚到迁移前的旧值没有业务意义，且无法安全区分：

- 哪些 `false` 是修复前误刷出来的错误值
- 哪些 `false` 是本就正确的“按环境独立配置”值

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

// 3. 对没有 nested envConfigMode 的 def，顶层 isUnifiedConfig 应与 defID 关联的环境实例数量一致
db.app_config_file_defs.aggregate([
  {
    $match: {
      envConfigMode: { $exists: false }
    }
  },
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
