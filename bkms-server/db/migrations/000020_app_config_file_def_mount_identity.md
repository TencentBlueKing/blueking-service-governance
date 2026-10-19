# app_config_file_defs：挂载路径改为 appID + mountDir + name 唯一

## 行为

- 去掉 `appID + name` 唯一索引。同一应用下，不同目录可以有同名文件。
- `appID + mountDir + name` 改为非 partial 唯一索引。`mountDir` 不落库时按空值参与唯一约束，Helm / Agones 的 values 文件名在同一应用下仍然唯一。
- 只有一条 framework def 的 trpc/taf 应用：把 `app_models` 上的 `fileName` / `filePath` 回填到该 def，并去掉末尾斜杠。回填成功后，从 `app_models` 去掉这两个字段。
- 有多条 framework def 的应用不回填、也不清 `app_models`。需要先跑框架 def 修复，再单独处理。
- `..` 这类路径只在后续写入时由 `path.Clean` 收齐，本迁移只去掉末尾 `/`。

## down

只恢复旧索引。已经回填的 def 和已经删除的 `app_models` 字段不会改回去。

若同一应用下已经存在同名文件，重建 `appID + name` 唯一索引会失败。
