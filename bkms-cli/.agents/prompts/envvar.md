# bkms-cli envvar Reference

`bkms-cli envvar` 用于管理环境变量的 CRUD、导入和导出。支持三种作用域，每种作用域对应独立的子命令：

| 作用域子命令 | 说明 | 定位方式 |
|---|---|---|
| `scoped` | 工作空间级、envType 级或 env 级 | `--scope`（默认 workspace）|
| `env` | 单个标准环境或特性环境，按工作空间内的环境名定位 | `--env <env-name>` |
| `app` | 应用级，仅在该应用下生效 | `--app <appID>` |

## list

列出环境变量。

```bash
# 列出工作空间下的 scoped 变量（含 workspace 级和 envType 级）
bkms-cli envvar list scoped

# 指定工作空间
bkms-cli envvar list scoped --workspace ws-demo

# 列出某个环境名下的变量（按 env-name 定位，内部自动解析 envID）
bkms-cli envvar list env --env staging

# 列出应用直接定义的变量
bkms-cli envvar list app --app myapp

# 列出应用在某环境下最终生效的全部变量（含继承）
bkms-cli envvar list app --app myapp --env prod

# JSON 格式
bkms-cli envvar list app --app myapp -o json

# 提取所有 key
bkms-cli envvar list app --app myapp -o 'jq=[.[] | .key]'
```

## create

创建环境变量。

```bash
# 创建工作空间级 scoped 变量（默认 workspace scope，所有环境生效）
bkms-cli envvar create scoped --key MY_KEY --value my-value

# 创建 envType 级 scoped 变量（仅在 production 类型环境生效）
bkms-cli envvar create scoped --key PROD_KEY --value prod-value --scope envType:production

# 创建环境名级变量（--env 为环境名，非 env ID）
bkms-cli envvar create env --env staging --key STAGING_KEY --value staging-value

# 创建应用级变量
bkms-cli envvar create app --app myapp --key APP_KEY --value app-value

# 创建敏感变量（值加密存储，list 时脱敏）
bkms-cli envvar create app --app myapp --key APP_CERT_CONTENT --value my-cert-value --sensitive

# 添加描述
bkms-cli envvar create app --app myapp --key MY_KEY --value val --description "My description"
```

## update

更新环境变量的值、key、描述或敏感属性。

```bash
# 更新应用级变量的值
bkms-cli envvar update app --app myapp --key MY_KEY --value new-value

# 重命名 key（--updated-key 指定新 key）
bkms-cli envvar update app --app myapp --key OLD_KEY --updated-key NEW_KEY

# 更新描述
bkms-cli envvar update app --app myapp --key MY_KEY --description "Updated description"

# 更新为敏感变量
bkms-cli envvar update app --app myapp --key MY_KEY --sensitive

# 取消敏感标记
bkms-cli envvar update app --app myapp --key MY_KEY --no-sensitive

# 更新环境名级变量（--env 为环境名）
bkms-cli envvar update env --env staging --key STAGING_KEY --value new-value

# 更新 scoped 变量（默认 workspace scope）
bkms-cli envvar update scoped --key MY_KEY --value new-value

# 更新 envType scoped 变量
bkms-cli envvar update scoped --key PROD_KEY --value new-value --scope envType:production
```

## delete

删除指定环境变量。

```bash
# 删除应用级变量
bkms-cli envvar delete app --app myapp --key APP_KEY

# 删除环境名级变量（--env 为环境名）
bkms-cli envvar delete env --env staging --key STAGING_KEY

# 删除工作空间级 scoped 变量
bkms-cli envvar delete scoped --key MY_KEY

# 删除 envType scoped 变量
bkms-cli envvar delete scoped --key PROD_KEY --scope envType:production
```

## export

将环境变量导出为 `.env` 文件格式（`KEY=value` 每行一条）。默认输出到 stdout，`-f` 写入文件。

```bash
# 导出应用自定义变量到 stdout
bkms-cli envvar export app --app myapp

# 导出应用在某环境下最终生效的全部变量（含继承，需指定 --scope effectiveByEnv --env）
bkms-cli envvar export app --app myapp --scope effectiveByEnv --env prod

# 导出到文件
bkms-cli envvar export app --app myapp --scope effectiveByEnv --env prod -f effective.env

# 导出环境名级变量（--env 为环境名）
bkms-cli envvar export env --env staging

# 导出 scoped 变量
bkms-cli envvar export scoped
```

`export app` 的 `--scope` 可选值：
- `appDefined`（默认）：仅导出应用直接定义的变量
- `effectiveByEnv`：导出在指定环境下最终生效的全部变量（需要 `--env`）

## import

从 `.env` 文件批量导入环境变量。支持 `--preview` 预览变更，不实际写入；`-o` 控制 preview 输出格式。

```bash
# 预览导入（不实际执行）
bkms-cli envvar import app --app myapp -f app.env --preview

# 预览并以 JSON 输出变更详情
bkms-cli envvar import app --app myapp -f app.env --preview -o json

# 实际导入应用级变量
bkms-cli envvar import app --app myapp -f app.env

# 导入环境名级变量（--env 为环境名）
bkms-cli envvar import env --env staging -f staging.env

# 导入 scoped 变量
bkms-cli envvar import scoped -f scoped.env
```

## scope 参数说明

`--scope` 参数适用于 `scoped` 子命令，格式为：

| 值 | 说明 |
|----|------|
| 省略（默认） | workspace 级，所有环境生效 |
| `envType:production` | 仅 production 类型环境生效 |
| `envType:staging` | 仅 staging 类型环境生效 |
| `envType:test` | 仅 test 类型环境生效 |
| `envType:development` | 仅 development 类型环境生效 |

## 特性环境

特性环境直接使用完整环境名（例如 `feat-myapp-1`），不需要 `--app`。以下操作使用当前默认工作空间，也可以通过 `--workspace` 显式指定。

```bash
bkms-cli envvar create env --env feat-myapp-1 --key TEST_MODE --value on
bkms-cli envvar list env --env feat-myapp-1
bkms-cli envvar update env --env feat-myapp-1 --key TEST_MODE --value off
bkms-cli envvar import env --env feat-myapp-1 -f vars.env --preview
bkms-cli envvar import env --env feat-myapp-1 -f vars.env
bkms-cli envvar export env --env feat-myapp-1 -f vars.env
bkms-cli envvar delete env --env feat-myapp-1 --key TEST_MODE
```
