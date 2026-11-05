# bkms-cli app image Reference

`bkms-cli app image` 用于查看应用镜像、从镜像仓库刷新镜像列表，以及晋级镜像 tag。

部署到生产环境的镜像 tag 必须先晋级，否则部署会被服务端拒绝（`has not been promoted to production`）。晋级不可撤销。

## 常见流程

平台构建成功后镜像会自动同步到镜像列表，确认 tag 后直接晋级，再部署到生产环境。

```bash
bkms-cli app image list --app demo --keyword v1.0.0
bkms-cli app image promote --app demo --tag v1.0.0
```

镜像由外部流水线等方式推送到仓库时，镜像列表不会自动更新，需要先刷新再晋级。晋级时服务端会校验 tag 是否在镜像列表中，不在则返回 404。

```bash
bkms-cli app image refresh --app demo
bkms-cli app image promote --app demo --tag v1.0.0
```

## list

查看应用镜像（最近 10 条），`isPromoted` 表示是否已晋级。

```bash
bkms-cli app image list --app demo
bkms-cli app image list --app demo --keyword v1.0 -o json
```

## refresh

从镜像仓库同步最新的 tag 列表。已有刷新在进行中时会提示稍后再查看，不视为失败。

```bash
bkms-cli app image refresh --app demo
```

## promote

晋级一个或多个镜像 tag。多个 tag 用逗号分隔或重复 `--tag`；逐个晋级，某个 tag 失败不影响其余 tag，只要有 tag 失败命令就以非零退出码结束。重复晋级已晋级的 tag 会把晋级人和晋级时间更新为本次操作。

```bash
bkms-cli app image promote --app demo --tag v1.0.0
bkms-cli app image promote --app demo --tag v1.0.0,v1.0.1
```
