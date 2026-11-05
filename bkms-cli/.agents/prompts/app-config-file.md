# 应用配置文件

`app config-file` 管理挂载到应用容器内的配置文件。配置文件分两类：

- `framework`：框架配置文件（Helm values、tRPC/TAF 配置），随应用创建，不能通过 CLI 新建。
- `plain`：用户自定义的文本文件，挂载到指定目录，容器内路径为 `<mount-dir>/<name>`，仅 trpc/taf/bkmsApp 应用支持。

文件新建后为统一配置（`ENV_MODE=unified`），所有环境共享默认内容。用 `edit --env` 修改某个环境时，
文件会自动切换为按环境独立配置（`per-env`），该环境得到自己的内容，其他环境继续使用默认内容。
Helm 类应用没有环境级配置，不能使用 `--env`。

定位文件用 `--name`；同名文件位于不同目录时加 `--mount-dir`，或直接用 `list` 输出中的 `--id`。
应用命令的 `--app` 支持应用 ID 或名称，`--workspace` 可使用已配置的默认值。

会删除数据的操作（`delete`、`reset`、`update --unified`、缩小 `update --mount-envs`）必须带 `--yes`，
未带时只打印将被删除的环境内容并退出。

## 常用流程

先查看应用有哪些配置文件，`MOUNT_ENVS` 为 `*` 表示挂载到所有环境。

```bash
bkms-cli app config-file list --app demo
```

新建一个 plain 文件，再查看某个环境下最终挂载了哪些文件。

```bash
bkms-cli app config-file create --app demo --name app.conf --mount-dir /data/conf -f ./app.conf
bkms-cli app config-file mount-preview --app demo --env prod
```

修改内容时，先用 `view --raw` 导出可编辑内容，本地修改后用 `edit` 写回。`--raw` 输出的字段与 `edit` 写入的字段一致。

```bash
bkms-cli app config-file view --app demo --name app.conf --raw > app.conf
bkms-cli app config-file edit --app demo --name app.conf -f app.conf --description "调整超时"
```

给某个环境单独配置内容，不影响其他环境；之后可以用 `reset` 让该环境恢复使用默认内容。

```bash
bkms-cli app config-file view --app demo --name app.conf --env prod --raw > app-prod.conf
bkms-cli app config-file edit --app demo --name app.conf --env prod -f app-prod.conf
bkms-cli app config-file list-env-instances --app demo --name app.conf
bkms-cli app config-file reset --app demo --name app.conf --env prod --yes
```

查看历史版本、对比差异并回滚。不带 `--env` 操作默认文件，带 `--env` 操作该环境自己的内容。

```bash
bkms-cli app config-file list-versions --app demo --name app.conf --env prod
bkms-cli app config-file diff-version --app demo --name app.conf --env prod --from 2
bkms-cli app config-file rollback-version --app demo --name app.conf --env prod --version 2
```

## list

列出应用下所有配置文件，包括 framework 和 plain。

```bash
bkms-cli app config-file list --app demo -o json
```

## create

创建 plain 文件。默认挂载到所有环境、内容为空、不启用环境变量渲染。

```bash
# 仅挂载到 dev 和 prod，并启用 ${{ env.KEY }} 环境变量渲染
bkms-cli app config-file create --app demo --name app.conf --mount-dir /data/conf \
  --file-content $'timeout=3s\n' --mount-envs dev,prod --enable-env-var-render
```

## update

只修改传入的参数。plain 可改名称、目录、挂载环境、环境模式和环境变量渲染；framework 可改环境模式和
`--env-content-strategy`（`overlay` 环境内容为补丁，`overwrite` 环境内容为完整文件），能否改名称、目录以后端规则为准。

```bash
bkms-cli app config-file update --app demo --name app.conf --new-name app.ini --new-mount-dir /etc/demo
bkms-cli app config-file update --app demo --name app.conf --per-env

# 以下操作会删除环境内容，需要 --yes
bkms-cli app config-file update --app demo --name app.conf --mount-envs dev --yes
bkms-cli app config-file update --app demo --name app.conf --unified --yes
```

`--mount-all-envs` 挂载到应用当前已有的所有环境，之后新建的环境不会自动挂载。

## delete

删除文件及其所有环境内容和版本历史。

```bash
bkms-cli app config-file delete --app demo --name app.conf --yes
```

## view

不带 `--env` 查看默认文件；带 `--env` 查看该环境实际生效的内容，`sharesDefault: true` 表示该环境没有自己的内容。

```bash
bkms-cli app config-file view --app demo --name app.conf --env prod -o json
bkms-cli app config-file view --app demo --name app.conf --env prod --raw > app-prod.conf
```

## edit

整体替换内容，每次写入生成一个新版本。`--file` 与 `--file-content` 二选一。
framework 文件在 overlay 策略下，环境内容是叠加到默认文件上的补丁，可用 `--view-compiled-content` 查看合并结果。

```bash
bkms-cli app config-file edit --app demo --name app.conf -f ./app.conf
bkms-cli app config-file edit --app demo --name trpc_go.yaml --env prod -f patch.yaml --view-compiled-content
```

## reset

删除某环境自己的内容及其版本历史，使其恢复使用默认内容。

```bash
bkms-cli app config-file reset --app demo --name app.conf --env prod --yes
```

## list-env-instances

列出默认文件及有独立内容的环境，未列出的环境使用默认内容。

```bash
bkms-cli app config-file list-env-instances --app demo --name app.conf
```

## mount-preview

查看某环境最终挂载到容器的文件。`CONTENT_SOURCE` 为 `normal`（默认内容）、`overlay`（补丁合并）或 `overwrite`（环境自己的完整内容）。

```bash
bkms-cli app config-file mount-preview --app demo --env prod
```

## 版本命令

`--version`（版本号）与 `--version-id`（`list-versions` 输出的记录 ID）二选一。
环境没有自己的内容时，`rollback-version`、`delete-version` 会拒绝执行，请去掉 `--env` 操作默认文件。

```bash
bkms-cli app config-file list-versions --app demo --name app.conf
bkms-cli app config-file view-version --app demo --name app.conf --version 3 -o 'jq=.content'

# 不带 --to 时与当前版本对比；带 -o 时输出两个版本的结构化数据
bkms-cli app config-file diff-version --app demo --name app.conf --from 2 --to 4

bkms-cli app config-file rollback-version --app demo --name app.conf --version 3 --description "回滚"
bkms-cli app config-file delete-version --app demo --name app.conf --version 1
```
