# app_config_file_defs：回填 envContentStrategy

## 背景

框架配置文件的环境内容策略改由定义上的 `envContentStrategy` 决定：

- `overlay`：环境内容是补丁，部署时与默认配置合并；
- `overwrite`：环境内容是完整配置，之后修改默认配置不影响已单独配置的环境。

新建 tRPC / TAF / bkmsApp 框架定义写入 `overwrite`。Helm / Agones 创建时写入 `overlay`。存量框架定义没有该字段，迁移前的行为都是与默认配置合并。

普通文本文件不使用这个字段，本 migration 不改它们。

## 迁移语句说明

`up` 是一条 `update` 命令，`multi: true`。

**筛选条件 `q`**

```json
{ "configKind": "framework", "envContentStrategy": { "$exists": false } }
```

只补还没有该字段的框架定义。新代码已经写入 `overlay` 或 `overwrite` 的文档跳过，迁移幂等。

**更新 `u`**

```json
{ "$set": { "envContentStrategy": "overlay" } }
```

回填值是常量，不用聚合管道。

先执行完本 migration，再让读取该字段的新二进制接流量。新代码不对缺失值做兜底，缺字段的框架定义在读取策略、切换策略或创建环境实例时都会报错。

## down

`$unset` 所有 `configKind: "framework"` 且已有 `envContentStrategy` 的文档。回滚后字段消失，与迁移前形状一致。无法区分哪些值是本 migration 回填的 `overlay`，哪些是新代码写入的 `overwrite`。

## 验证

```js
db.app_config_file_defs.countDocuments({
  configKind: "framework",
  envContentStrategy: { $exists: false }
})
```

结果应为 0。
