`migrations` 子目录中存放所有 golang-migrate/migrate 工具所需要的数据库迁移文件。

- 使用 `./bin/migrate create -ext json -seq -dir db/migrations some_model_idx` 命令来创建新的 migration；
- 每个迁移文件为 json 格式，格式为 MongoDB 原生 Database Command；
- 每增加一对 json 文件后，维护一个与之对应的 markdown 文件作为文档；
- seq 序号不得与目标分支已有的 migration 重复，重号会导致后合入的那份被永久跳过且不报错；
