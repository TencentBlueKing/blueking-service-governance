# app_config_files 补回填 defID

修复 000011 迁移遗留问题：环境级配置文件缺少 `defID` 字段。

## 背景

000011 迁移的第二条 update 规则要求环境级文档上存在 `defaultAppConfigFileID`
字段才能回填 `defID`。但 framework 类型的环境级配置文件从未写入过该字段
（它们通过 `appID` + `name` 隐式关联默认配置），导致这些记录的 `defID`
未被回填。

## 修复策略

通过聚合管道对每条缺少 `defID` 的环境级配置文件，按 `appID` + `configKind` 查找
同应用同类型的默认配置记录（`envName` 为空），以其 `_id` 作为 `defID` 回填。

## 影响

- 仅影响 `defID` 为空的环境级配置文件记录
- 已有 `defID` 的记录不受影响
