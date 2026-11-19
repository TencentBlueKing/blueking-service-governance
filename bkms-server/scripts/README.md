# scripts

本目录存放 bkms-server 相关的辅助脚本，包括构建、运维、数据处理等一次性或周期性任务。
业务运行时使用的代码不放在这里，应放到 `cmd/` 或对应的业务包中。

本目录下的 Python 脚本统一采用 [PEP-723](https://peps.python.org/pep-0723/) 内联元数据声明 Python 版本与依赖，
推荐直接使用 [`uv`](https://docs.astral.sh/uv/) 执行（无需手动创建虚拟环境）：

```bash
uv run scripts/<category>/<script-name>.py [参数...]
```

具体约定可参阅 [AGENTS.md](./AGENTS.md)。

## 目录组织约定

新增脚本请**按场景归类到子目录**中组织，方便通过目录结构快速识别脚本所属场景：

- 目录与文件名统一使用短横线（`-`）风格，不使用下划线
- 分类子目录采用小写英文，形如 `scripts/<category>/<script-name>.<ext>`
- 单一脚本、暂无同类其它脚本的场景，可以直接放在 `scripts/` 根目录；一旦出现第二个同类脚本，应新建子目录收纳
- 目前已有的分类子目录：
    - `platform-build/`：与"平台构建"相关的脚本，例如准备构建所需的运行时基础镜像清单、将镜像同步到内部仓库等

后续如需引入其他分类（例如数据迁移、故障排查等），请沿用此规则，并在下方"脚本清单"章节补充说明。

## 脚本清单

### `platform-build/` —— 平台构建

这一组脚本用于维护"平台构建"能力所依赖的运行时基础镜像：先由 `gen-runtime-images.py` 生成经过筛选的镜像清单文件，
再由 `sync-runtime-images.py` 将清单中的镜像同步到内部镜像仓库，供平台构建流程消费。

#### `platform-build/gen-runtime-images.py`

- **用途**：从 `golang` 官方镜像的 tag 清单中筛选出符合平台使用要求的精确小版本 tag
  （形如 `golang:X.Y.Z` 或 `golang:X.Y.Z-alpineA.B`），并生成用于镜像同步的清单文件。
- **过滤规则**：只保留三段式精确版本；只保留 go 大版本 `>= 1.20`；Debian 系精确小版本会直接保留，
  同一 go patch 的 alpine 精确小版本仅保留其中最新的 alpine 版本，并从最终结果中反向提取所需的 `alpine:A.B` 基础镜像。
- **数据源**：支持两种，二选一
    - `--source skopeo`：直接调用 `skopeo list-tags docker://golang` 拉取（需本机安装 `skopeo`）
    - `--source file --input <path>`：从本地 tag 清单文件读取
- **产出**：默认写入 `configs/workload-runtime-images.txt`，可通过 `--output` 覆盖。
- **使用示例**：

    ```bash
    # 从本地文件读取，生成默认位置的清单
    uv run scripts/platform-build/gen-runtime-images.py \
        --source file --input golang-alpine.txt

    # 直接调用 skopeo 拉取远端 tag
    uv run scripts/platform-build/gen-runtime-images.py --source skopeo
    ```

#### `platform-build/sync-runtime-images.py`

- **用途**：读取镜像清单文件，将其中的镜像从 Docker Hub 批量同步到指定的目标仓库（`pull → tag → push`）。
- **依赖**：本机已安装 `docker`，并已 `docker login` 目标仓库；Python 侧零第三方依赖。
- **行为要点**：
    - 目标仓库前缀末尾多余的 `/` 会被自动裁剪
    - 目标已存在的镜像会主动跳过（通过 `docker manifest inspect` 探测），避免重复拉取和推送
    - 单个镜像失败不影响后续镜像，最终统一汇总"成功 / 跳过 / 失败"数量
    - `golang` 镜像强制要求使用 `X.Y.Z` 或 `X.Y.Z-alpineA.B` 精确小版本，防止误同步滚动 tag
    - 支持 `--dry-run` 只打印命令、不实际执行
- **使用示例**：

    ```bash
    # 正式同步
    uv run scripts/platform-build/sync-runtime-images.py \
        --target-registry mirrors.tencent.com/bkms \
        --input configs/workload-runtime-images.txt

    # 预览要执行的命令，不真正执行
    uv run scripts/platform-build/sync-runtime-images.py \
        --target-registry mirrors.tencent.com/bkms \
        --input configs/workload-runtime-images.txt \
        --dry-run
    ```

## 典型工作流

以"新增平台构建可用的 Go 运行时版本"为例：

1. 运行 `platform-build/gen-runtime-images.py`，重新生成 `configs/workload-runtime-images.txt`
2. 人工 review 该清单文件的 diff，确认新增/剔除的 tag 符合预期
3. 运行 `platform-build/sync-runtime-images.py` 将清单中的镜像同步到内部仓库
4. 提交清单文件的变更
