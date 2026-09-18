# 蓝盾 devops 社区版对接 · 待办事项

> 仅记录尚未完成、需后续跟进的事项。

## TODO 1：`CreateRepository` 按代码库类型参数化认证字段

**现状**：`pkg/infras/cloudapi/bkci/bkci.go` 的 `CreateRepository` 仅支持传 `@type`（`repoType`），
`authType` 写死 `"OAUTH"`、`credentialId` 写死 `""`、无 `scmCode` 字段（代码中已标 TODO）。

**要做**：

1. 扩展 `CreateRepository`（`client.go` 接口 + `bkci.go`/`stub.go` 实现）支持 `authType`、
   `credentialId`、`scmCode` 字段参数化，按 `@type`（SVN/GitLab/GitHub/TGit/P4）构造不同请求体。
2. 凭证由上层先调用 `CreateCredential`（USERNAME_PASSWORD）/`CreateAccessTokenCredential`（ACCESSTOKEN）
   创建，再把凭证 ID 传入 `CreateRepository`。

**依赖/阻塞**：`scmCode` 字段的具体取值/含义待确认（疑为 `scmType` 枚举值，如 `CODE_SVN`/`CODE_GITLAB`）。

## TODO 2：上层链路改造（社区版下代码库接入改为手动方式）

**现状**：

- `pkg/bkintegrations/bkci/manager.go` 的 `createRepository` 固定传 `bkci.RepoTypeCodeGit`（已留注释）。
- `pkg/bkintegrations/handler/bkci.go` 仍在调用社区版无接口的方法：
  - `ListOAuthGitProjects`（:66）、`GetOAuthUrl`（:112）；
  - `ListRepositoryBranches`（:305）、`ListRepositoryTags`（:360）。

**要做**：

1. `manager.go` 的 `createRepository`/`Initialize` 增加代码库类型入参，按实际类型传对应 `@type` 常量，
   并配套 TODO 1 的 `authType`/`credentialId` 传入。
2. `handler/bkci.go` 中 4 个调用点在 `config.G.Community.Devops == true` 时改为手动填写流程：
   - 去掉 OAuth 授权交互，改为按类型输入仓库地址 + 凭证；
   - 去掉分支/标签下拉拉取，改为用户手动输入分支/标签名。

**依赖/阻塞**：属总体方案「八、按链路补充社区版信息」范围，需前端（bkms-ui）配合改造表单交互。
