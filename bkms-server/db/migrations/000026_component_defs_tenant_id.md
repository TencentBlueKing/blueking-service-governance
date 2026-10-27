# component_defs：按租户划分，实例计数拆表

## 背景

`component_defs` 接入 store 层租户过滤，平台租户中的内置定义共享给所有租户读取，方案见
`design_notes/component_def_tenancy.md`。内置定义被多个租户共用，实例计数不能再放在定义上，
改为按租户划分的 `component_def_instance_counts`。**本迁移必须先于新代码上线。**

存量环境都是单租户，数据全部归属 `default`；单租户模式下平台租户也是 `default`，内置定义无需搬迁。

## up

1. 回填：`tenantID` 缺失的定义写入 `default`，可重复执行。
2. 唯一索引：新建 `tenantID_1_name_1_version_1`，再删 `name_1_version_1`。先建后删，中间不会出现没有唯一约束的窗口。
3. 计数表：建唯一索引 `tenantID_1_defName_1_version_1`（同时创建集合），
   再把定义上非零的 `appCompInstanceCount`、`workspaceCompInstanceCount` 通过 `$merge` 写入计数表，已有记录不覆盖。
4. 从定义上 `$unset` 两个计数字段。

## down

1. 计数表通过 `$merge` 写回定义上的计数字段，再删除计数表。
2. 重建 `name_1_version_1`，删除 `tenantID_1_name_1_version_1`。不同租户存在同名定义时重建会失败，需要先人工处理。
3. `$unset` 值为 `default` 的 `tenantID`；其它租户的值不回滚。

## 发布顺序

1. 执行本迁移。
2. 发布新版本代码。
3. 全部实例升级完成后：
   - 再回填一次 `tenantID`，旧 Pod 退出前创建的定义不带该字段：
     ```js
     db.component_defs.updateMany({ tenantID: { $exists: false } }, { $set: { tenantID: "default" } })
     ```
   - 再执行一次 `load_builtin_component`，确保内置定义都带 `isBuiltin: true`。

## 验证

```js
db.component_defs.countDocuments({ tenantID: { $exists: false } })              // 应为 0
db.component_defs.getIndexes()
db.component_def_instance_counts.getIndexes()
```

混跑窗口内旧 Pod 写在定义上的计数增量不搬迁：窗口很短，期间增删组件的概率低，偏差可接受；
定义上残留的计数字段新代码不读取。
