# bkms-cli update Reference

`bkms-cli update` 用于检查并安装 bkms-cli 的新版本，执行时不需要登录 BKMS。

`github` 更新源从 GitHub Releases 查找当前平台的 `bkms-cli` 制品，并使用 Release 中的 `checksums.txt` 校验下载内容。`repo` 更新源从构建时指定的 `latest` 目录读取 `version` 纯文本文件，再下载当前平台对应的二进制，并使用响应头 `X-Checksum-Sha256` 校验。

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

GitHub 发布 tag 必须使用 `vX.Y.Z` SemVer 格式。每个平台的资产名必须与下面的 `bkms-cli-{os}-{arch}` 格式完全匹配，因此不会误选同仓库里 `bkms-server` 等其他产品的制品。

## repo 制品目录

```text
latest/
├── version
├── bkms-cli-linux-amd64
├── bkms-cli-linux-arm64
├── bkms-cli-darwin-amd64
├── bkms-cli-darwin-arm64
├── bkms-cli-windows-amd64.exe
└── bkms-cli-windows-arm64.exe
```

`version` 只包含一行 SemVer，例如 `v1.3.0`。发布时先上传全部平台二进制，最后覆盖 `version` 文件。
