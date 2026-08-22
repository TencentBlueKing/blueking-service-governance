# bkms-cli config Reference

管理本地配置文件（默认 `~/.bkms/config.yaml`，可用 `BKMS_CLI_CONFIG` 覆盖）。

## 查看配置

```bash
bkms-cli config view
```

## 设置 API 地址

```bash
bkms-cli config set --bkms-base-url https://bkms.example.com
bkms-cli config set --bcs-api-host https://bcs-api.example.com
bkms-cli config set \
  --bkms-base-url https://bkms.example.com \
  --bcs-api-host https://bcs-api.example.com
bkms-cli config set --if-unset --bkms-base-url https://bkms.example.com
```

至少提供一个 flag；未传的字段保持不变。写入前会去掉 URL 尾部 `/`。

`--if-unset`：仅当配置文件中该字段为空时才写入（npm postinstall 用这个，避免更新时覆盖已有配置）。

`bkmsBaseUrl` 未配置时，`bkms-cli`（无参）、`login` 以及需鉴权的业务命令会提示先执行 `config set`。
