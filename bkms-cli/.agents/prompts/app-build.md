# bkms-cli app build Reference

`bkms-cli app build` 用于触发应用镜像构建，以及查看构建记录。

创建构建时，`--branch`、`--tag`、`--commit` 三选一，并且必须提供 `--image-tag`。CLI 不计算推荐版本号。

## create

按分支、Tag 或 CommitID 触发一次镜像构建。

```bash
# 按分支构建。已有脚本继续使用 --branch 即可
bkms-cli app build create --app demo --branch main --image-tag v1.0.0

# 按 Tag 构建
bkms-cli app build create --app demo --tag v1.0.0 --image-tag v1.0.0

# 按 CommitID 构建。CommitID 为 7–40 位十六进制
bkms-cli app build create --app demo --commit a1b2c3d --image-tag v1.0.0
```

三个来源都不给，或同时给了多个，命令会报错并退出，不会发起构建请求。`--commit` 格式不合法时同样在本地报错，不发起请求。

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--branch` | 三选一 | 按分支构建 |
| `--tag` | 三选一 | 按 Tag 构建 |
| `--commit` | 三选一 | 按 CommitID 构建，7–40 位十六进制 |
| `--image-tag` | 是 | 本次构建使用的镜像 Tag |

## list

查看应用的构建记录。

```bash
bkms-cli app build list --app demo
```
