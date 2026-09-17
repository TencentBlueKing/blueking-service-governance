# 000032 code_repositories

代码库改由 bkms 持有。

up：

1. 创建 `code_repositories` 索引：唯一 `workspaceID + alias`，普通 `tenantID`。
2. `bkci_repositories` 存量记录写入 `code_repositories`，接入方式为 `TGit`，已存在的不覆盖。
3. 代码库配置 `type` 的历史取值 `GitHub` 改为 `TGit`：
   - `build_configs.codeRepo.type`
   - `applications.helmSpec.helmSource.gitRepoConfig.type`

down：删除 `code_repositories`，`GitHub` 不回滚。
