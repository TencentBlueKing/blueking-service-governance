---
name: bkms-dev-ginapi
description: "在 bkms-server 中开发新的 Gin REST API 时使用；聚焦 router、handler、serializer、鉴权、错误处理、Swagger 与测试约定。"
---

你正在 `apps/bkms-server` 中工作，目标是新增或扩展一个基于 Gin 的 REST API。开始前先阅读相关领域代码、现有 router / handler / serializer、领域模型与测试，避免脱离现有实现风格。

## 核心规则

- 路由注册放在模块的 `router.go`，handler 文件只放视图逻辑。
- Handler 应是带依赖的结构体方法；当前默认直接注入 `*storereg.Registry`（`pkg/server/registry`），
  需要模型的 Store 时用 `store.Of(h.registry.DB, XxxMeta)`，见 `design_notes/generic_store.md`。
- 注意 import cycle。安全模式是：
  - `pkg/<domain>/router.go` 只定义路由和小型 handler interface。
  - `handler.New(registry)` 在上层完成构造后，再调用 `domain.Register(...)`。
- 所有可复用的 Gin 工具优先放在 `pkg/bkmssrv/ginutils`。
  - 鉴权放在 `pkg/common/auth`
  - 错误渲染放在 `pkg/common/bkerrs`
  - 可复用权限 / 路径资源校验放在 `pkg/bkmssrv/ginutils/perm`

## 推荐结构

```text
pkg/<domain>/
  router.go
  handler/
    <feature>.go
  serializer/
    <feature>.go
```

`router.go` 示例：

```go
type Handler interface {
    Create(c *gin.Context)
    Update(c *gin.Context)
}

func Register(rg *gin.RouterGroup, h Handler) {
    rg.POST("/...", h.Create)
    rg.PUT("/...", h.Update)
}
```

## Handler 约定

Handler 推荐结构：

```go
type Handler struct {
    registry *storereg.Registry
}

func New(registry *storereg.Registry) *Handler {
    return &Handler{registry: registry}
}
```

单个 handler 方法通常按这个顺序组织：

1. 用 `ginutils.BindURI` 绑定路径参数。
2. 有 body 时用 `ginutils.BindJSON` 绑定请求体。
3. 用 `ginutils/perm` 做可复用的权限和资源校验。
4. 处理当前接口特有的业务校验。
5. 通过 `store.Of(h.registry.DB, XxxMeta)` 或领域 service 读写数据。
6. 把领域模型转换为 serializer output。
7. 用 `ginutils.JSON` 或 `ginutils.OK` 返回成功响应。
8. 出错时用 `bkerrs.AbortErr(c, err)`。

不要为了“分层”额外包一层没有复用价值的私有函数。请求到响应的主流程应保持直接、清晰。

## Serializer 与校验

- serializer 放在 `pkg/<domain>/serializer`。
- 命名优先使用 `Input` / `Output`，不要使用 `Request` / `Response`。
- JSON 字段名应和现有 API 保持一致。
- 优先使用 Gin / validator 的 `binding` tag 做静态校验，例如 `required`、`oneof`、自定义 validator。
- 自定义 validator 只做字段本地校验；涉及数据库、权限、workspace 上下文的校验留在 handler 或领域逻辑中。
- 路径参数可以单独定义 URI input 结构体，不要把所有路径校验都塞进业务 serializer。
- 如果字段在旧接口中需要以 string 编码来避免 int64 精度丢失，继续保留该行为，例如 `json:"bkCCBizID,string"`。
- output 转换逻辑应放在 serializer output 的转换方法中，并写清楚兼容语义。

## 鉴权与错误处理

- Gin 鉴权使用 `pkg/common/auth` 中的能力，常见链路是 `auth.UserAuth()`。
- Gin 错误处理中间件使用 `bkerrs.ErrorHandler()`。
- handler 内返回错误时使用 `bkerrs.AbortErr(c, err)`。
- Swagger 错误响应 schema 统一使用 `bkerrs.GinErrorOutput`。

## Swagger

每个 handler 方法都应补充 Swagger 注解，至少包含：

- `@ID`（接口唯一标识，使用 PascalCase 命名，如 `ListBCSAuthorizedProjects`）
- `@Summary`
- `@Tags`
- `@Accept`（仅 JSON body 接口需要）
- `@Produce json`
- `@Security BkUserInfo`
- `@Security BkUserCredential`
- `@Param`
- `@Success`
- `@Failure`（推荐只声明 400 和 404 错误码；不要声明 5xx 错误码，5xx 由框架统一处理）
- `@Router`

`@Router` 路径使用 `{param}` 占位符风格，且不包含服务公共前缀。

接口定义变化后运行 `make apidocs`。

## 测试与验证

- 为新增 API 补充对应的 Go 单元测试或 API 测试。
- Ginkgo 的 `Describe`、`Context`、`It` 文案必须使用英文。
- 响应测试优先先断言状态码，再断言完整 JSON；避免只检查 substring。
- 完成 Go 代码修改后，按仓库约定运行 `make lint`。
- 按需运行相关 `ginkgo` 测试与 Bruno API 测试。

交付前确认：

- 没有引入 proto 生成类型依赖。
- router / handler / serializer 分层清晰。
- 参数校验放置合理。
- Swagger 已更新。
- 相关测试已通过。
