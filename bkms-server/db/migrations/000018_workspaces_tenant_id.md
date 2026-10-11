# workspaces 补齐 tenant_id

`workspaces` 改为按租户访问。已有文档没有 `tenant_id`，先回填单租兼容值 `default`，再为该字段建普通索引。

- 回填：`tenant_id` 不存在的文档设为 `default`。
- 索引：`tenant_id_1`，供按租户过滤。
- `id` 仍保持全局唯一，本轮不改成租户内唯一。
