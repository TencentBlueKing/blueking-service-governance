# app_config_file_defs：回填 enableEnvVarRender

## 背景

`AppConfigFileDef` 新增环境变量渲染开关 `enableEnvVarRender`：

- framework：创建时恒为 `true`，且不允许通过 def 接口关闭；
- plain：创建时默认为 `false`，用户可改。

存量文档都是 framework，拆表迁移（000011）写入 def 时没有这个字段。Go 侧 `bool` 缺键解成 `false`，列表/详情会显示「未开启渲染」，与策略不符。TAF/tRPC 当前仍无条件走 `cfgrender`，部署暂时不受影响；一旦 framework 路径也读这个开关，缺字段的存量会突然不渲染。

新建 def 在 `createDef` 里总会写入该布尔值（bson 无 `omitempty`），因此「字段不存在」可以当作存量标记，不必再按 `configKind` 过滤。

## 迁移语句说明

`up` 是一条 `update` 命令，`multi: true`。

**筛选条件 `q`**

```json
{ "enableEnvVarRender": { "$exists": false } }
```

只补还没有该字段的文档。新代码创建的 framework（已是 `true`）和 plain（已是 `false`）一律跳过，迁移幂等。不要整表 `$set: true`，否则会把已显式关闭渲染的 plain 刷掉。

**更新 `u`**

```json
{ "$set": { "enableEnvVarRender": true } }
```

写入 framework 缺省值。回填值是常量，不用聚合管道。

## down

只 `$unset` `configKind: "framework"` 且 `enableEnvVarRender: true` 的文档。不能按「值为 true」无条件 unset：用户把 plain 打开渲染后，回滚会把业务选择清掉。

回滚后 framework 再次缺字段，解成 `false`，与迁移前形状一致。新建且尚未被 down 扫到的 plain 保持原样。

## 验证

```js
// 应为 0：不应再有文档缺少 enableEnvVarRender
db.app_config_file_defs.countDocuments({ enableEnvVarRender: { $exists: false } })

db.app_config_file_defs.aggregate([
  { $group: { _id: { kind: "$configKind", render: "$enableEnvVarRender" }, count: { $sum: 1 } } }
])
```

滚动发布窗口里旧 Pod 仍可能写出无该字段的 def；补跑同一条 `up` 即可。补齐前 TAF/tRPC 部署行为与迁移前相同。
