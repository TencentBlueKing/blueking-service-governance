# 空间框架配置模板

创建 tRPC、TAF 应用时，框架配置文件的初始内容来自模板。工作空间可以维护自己的模板，
并为每个框架、语言指定默认模板。代码位于 `pkg/workload/appmodelcore/frameworktmpl`。

模板只用于创建应用时填充框架配置文件：前端选择模板后把内容和文件名填入
`frameworkConfigFile`，创建接口与配置文件的存储都不感知模板。因此修改、删除模板不影响已创建的应用。

## 模板分组

模板按「框架 + 语言」分组，分组取自运行时能力目录中支持框架配置文件的组合：

| 框架 | 语言 | 内置文件名 | 格式 |
| --- | --- | --- | --- |
| trpc | go | `trpc_go.yaml` | yaml |
| trpc | cpp | `trpc_cpp.yaml` | yaml |
| taf | cpp | `config.conf` | taf |

TAF 只支持 cpp，创建模板时语言可以不填，由后端补全为 `cpp`。这样所有分组共用同一套唯一约束。

## 平台内置模板与空间自定义模板

- 内置模板放在 `frameworktmpl/builtin/`，通过 `go:embed` 打进二进制，每个分组恰好一个，
  名称固定为「平台默认」，ID 为 `builtin-{framework}-{language}`。内置模板只读，不入库。
- 自定义模板存储在 `workspace_framework_templates` 集合，列表接口把两者拼接返回，
  每个分组先列内置模板，再按更新时间倒序列自定义模板。

## 默认模板

默认状态只记录在自定义模板的 `isDefault` 字段上：

- 分组内至多一个自定义模板 `isDefault=true`，由部分唯一索引保证；
- 没有自定义默认模板时，内置模板就是默认，因此删除默认模板后自动回落到内置模板；
- 设置默认时先清除分组内其它模板的标记，再给目标置位，不使用事务。并发设置时第二个请求会被
  部分唯一索引拒绝并返回冲突错误；
- 对内置模板调用「设为默认」表示清除分组内的自定义默认；
- 默认状态不属于模板内容，切换默认不刷新 `updatedAt`。

## 接口

均挂在 `/workspaces/{workspaceID}/framework-templates` 下，查询需要空间查看权限，写操作需要空间编辑权限：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `?framework=&language=` | 列表，不返回模板内容 |
| GET | `/{templateID}` | 详情（含内容），支持内置模板 ID |
| POST | `` | 新建，可同时设为默认 |
| PUT | `/{templateID}` | 编辑名称、文件名、内容与默认状态，框架与语言不可改；内置模板返回 400 |
| DELETE | `/{templateID}` | 删除；内置模板返回 400 |
| PUT | `/{templateID}/default` | 设为默认，支持内置模板 ID |

## 校验

- 名称：去掉首尾空白后 1~64 个字符，不能为「平台默认」，同一分组下不重复；
- 文件名：规则与应用配置文件名一致（字母、数字、`_`、`-`、`.`），最长 64；
- 内容：不能为空，最大 64KB，按分组格式校验（复用 `appcfg.FrameworkPolicy.ValidateContent`）。

## 审计与清理

- 写操作记录审计，资源类型为 `frameworkTemplate`，资源 ID 为模板 ID，并带上工作空间 ID；
  切换默认的审计记录不包含文件内容；
- 删除工作空间前通过 `hooks.RegisterPreDeleteHooks` 清理该空间的全部模板。
