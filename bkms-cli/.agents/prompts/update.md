# bkms-cli update Reference

`bkms-cli update` 用于检查并安装 bkms-cli 的新版本，执行时不需要登录 BKMS。

## 检查更新

```bash
bkms-cli update --check
```

如果有新版本，命令只输出当前版本和最新版本，不会替换当前程序。

## 安装更新

```bash
bkms-cli update
```

只有远端 SemVer 严格高于当前版本时，命令才会下载并替换二进制。校验失败、无写入权限或替换失败时，命令返回错误。

GitHub 发布 tag 和 repo 的 `version` 文件使用 `bkms-cli/vX.Y.Z` 格式。发布资产名使用 `bkms-cli-{os}-{arch}-{version}` 格式。

## 错误语义

- `ErrUpdateNotConfigured`：当前构建没有有效的更新源。
- `ErrInvalidVersion`：当前或远端版本不是有效的 SemVer。
- `ErrNoRelease`：GitHub Release 中没有当前平台可用的资产。
- `ErrBinaryTooLarge`：下载的更新资产超过大小限制。
- `ErrChecksumMissing`、`ErrChecksumInvalid`：repo 更新资产缺少有效的 SHA256 校验值。
