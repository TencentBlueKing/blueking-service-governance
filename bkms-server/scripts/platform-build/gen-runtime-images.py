#!/usr/bin/env python3
# -*- coding: utf-8 -*-

# TencentBlueKing is pleased to support the open source community by making
# 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
# Copyright (C) Tencent. All rights reserved.
# Licensed under the MIT License (the "License"); you may not use this file except
# in compliance with the License. You may obtain a copy of the License at
#
#  http://opensource.org/licenses/MIT
#
# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
# either express or implied. See the License for the specific language governing permissions and
# limitations under the License.
#
# We undertake not to change the open source license (MIT license) applicable
# to the current version of the project delivered to anyone in the future.

# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""
gen-runtime-images.py

用途：
    从 golang / python 官方镜像的 tag 清单中筛选出符合平台使用要求的精确小版本
    tag（形如 golang:X.Y.Z、golang:X.Y.Z-alpineA.B、python:X.Y.Z-slim），
    并生成一份用于镜像同步的清单文件。

支持两种输入源（二选一，必须显式指定）：
    - --source skopeo：直接调用 `skopeo list-tags docker://golang` 与 `docker://python` 拉取
    - --source file --golang-input <path> --python-input <path>：分别从本地文件读取两份 tag 清单

golang 过滤规则：
    1. 只保留 X.Y.Z 或 X.Y.Z-alpineA.B 形式的三段式精确版本
       - 丢弃：滚动 tag（如 1.25、1.25.5-alpine、1.20-alpine3.19）
       - 丢弃：预发布/开发版本（tip-、rc-、beta、pre 等）
       - 丢弃：无 go 版本号的纯 alpine tag
    2. 只保留 go 大版本 >= 1.20 的 tag
    3. 对同一个 go 小版本（X.Y.Z），若存在多个 alpine 版本，仅保留其中最新的一个
       - 例如 1.25.6-alpine3.22 与 1.25.6-alpine3.23，仅保留 1.25.6-alpine3.23
    4. Debian 系精确小版本（X.Y.Z）与 alpine 精确小版本会同时保留

python 过滤规则（作为 python 平台通用构建的运行镜像）：
    1. 只保留 X.Y.Z-slim 形式的精确版本：基于 Debian、自带 pip，且不受 PEP 668 限制
       - 丢弃：滚动 tag（如 3.12-slim）、预发布版本（如 3.14.0rc1-slim）
       - 丢弃：带发行版代号（如 -slim-bookworm）、alpine、windows 等其他变体
    2. 只保留 python 大版本 >= 3.10 的 tag

产出：
    脚本会将镜像清单写入 --output 指定的文件（默认：configs/workload-runtime-images.txt）。
    输出文件内容包含：
      - golang 系列：过滤后的所有 golang:X.Y.Z 与 golang:X.Y.Z-alpineA.B
      - alpine 系列：从 golang:X.Y.Z-alpineA.B 中反向提取的所有独立 alpine:A.B
      - python 系列：过滤后的所有 python:X.Y.Z-slim
    输出文件的父目录必须已经存在，否则直接报错退出。

使用示例：
    # 从本地文件读取，生成默认位置的清单
    uv run scripts/platform-build/gen-runtime-images.py \\
        --source file --golang-input golang-alpine.txt --python-input python.txt

    # 直接调用 skopeo 拉取远端 tag
    uv run scripts/platform-build/gen-runtime-images.py --source skopeo

    # 指定输出路径
    uv run scripts/platform-build/gen-runtime-images.py \\
        --source skopeo --output configs/workload-runtime-images.txt
"""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from typing import Iterable

# 匹配形如 "1.25.6" 或 "1.25.6-alpine3.23" 的精确三段式版本 tag
# 分组：go_major, go_minor, go_patch, alpine_major, alpine_minor
TAG_PATTERN = re.compile(
    r"^(\d+)\.(\d+)\.(\d+)(?:-alpine(\d+)\.(\d+))?$"
)

# 最低支持的 go 版本（含），例如 (1, 20) 表示 1.20 起
MIN_GO_VERSION = (1, 20)

# 匹配形如 "3.12.7-slim" 的 python 精确版本 tag，分组：major, minor, patch
PYTHON_TAG_PATTERN = re.compile(r"^(\d+)\.(\d+)\.(\d+)-slim$")

# 最低支持的 python 版本（含）：3.9 已停止维护，poetry / uv 的新版本也陆续不再支持
MIN_PYTHON_VERSION = (3, 10)

# 默认输出路径，产出的镜像清单文件用于镜像同步脚本消费
DEFAULT_OUTPUT_PATH = "configs/workload-runtime-images.txt"


def read_tags_from_file(path: str) -> list[str]:
    """从本地文件读取 tag 清单，每行一个 tag。

    文件行内容可能带有引号、逗号、空格（例如从 skopeo 的 JSON 输出拷贝过来），
    这里统一做剥离，只提取干净的 tag 字符串。
    """
    tags: list[str] = []
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            tag = line.strip().strip(",").strip('"').strip()
            if tag:
                tags.append(tag)
    return tags


def read_tags_from_skopeo(
    image: str = "docker://golang",
) -> list[str]:
    """通过 skopeo list-tags 命令直接获取 tag 清单。

    skopeo 返回 JSON，形如 {"Repository":"...","Tags":["1.0",...]}。
    这里返回完整 tag 列表，后续由 filter_and_pick_latest 统一筛选符合平台规则的精确小版本。
    """
    import json

    result = subprocess.run(
        ["skopeo", "list-tags", image],
        check=True,
        capture_output=True,
        text=True,
    )
    payload = json.loads(result.stdout)
    return payload.get("Tags", []) or []


def filter_and_pick_latest(tags: Iterable[str]) -> list[str]:
    """按规则过滤 tag，同时保留 Debian 与 Alpine 精确小版本。

    实现思路：
    1. 用正则匹配三段式精确 tag，天然排除 rc/beta/tip/滚动版本
    2. 校验 go 大版本 >= MIN_GO_VERSION
    3. Alpine tag 按 (go_major, go_minor, go_patch) 分组，只保留 alpine 版本最大的一个
    4. Debian tag 直接按 go 版本升序保留
    """
    # key: (go_major, go_minor, go_patch)
    # value: (alpine_major, alpine_minor, original_tag)
    best_alpine_per_patch: dict[tuple[int, int, int], tuple[int, int, str]] = {}
    debian_tags: dict[tuple[int, int, int], str] = {}

    for tag in tags:
        m = TAG_PATTERN.match(tag)
        if not m:
            continue

        go_major, go_minor, go_patch = int(m.group(1)), int(m.group(2)), int(m.group(3))
        if (go_major, go_minor) < MIN_GO_VERSION:
            continue

        key = (go_major, go_minor, go_patch)
        alpine_major, alpine_minor = m.group(4), m.group(5)
        if alpine_major is None or alpine_minor is None:
            debian_tags[key] = tag
            continue

        current = (int(alpine_major), int(alpine_minor), tag)

        # 同一个 go patch，比较 alpine 版本，保留更新的
        exist = best_alpine_per_patch.get(key)
        if exist is None or (current[0], current[1]) > (exist[0], exist[1]):
            best_alpine_per_patch[key] = current

    # 先输出 alpine 精确小版本，再输出 Debian 精确小版本，保持清单结构稳定
    alpine_selected = sorted(best_alpine_per_patch.items(), key=lambda kv: kv[0])
    debian_selected = sorted(debian_tags.items(), key=lambda kv: kv[0])
    return [f"golang:{item[1][2]}" for item in alpine_selected] + [
        f"golang:{item[1]}" for item in debian_selected
    ]


def filter_python_tags(tags: Iterable[str]) -> list[str]:
    """按规则过滤 python tag，返回按版本升序排列的 python:X.Y.Z-slim。

    只认 X.Y.Z-slim 一种变体：正则天然排除滚动 tag、预发布版本与其他发行版变体，
    每个精确版本至多一条，因此不需要像 golang alpine 那样再按发行版去重。
    """
    # key: (major, minor, patch)，用于排序与去重
    selected: dict[tuple[int, int, int], str] = {}

    for tag in tags:
        m = PYTHON_TAG_PATTERN.match(tag)
        if not m:
            continue

        major, minor, patch = int(m.group(1)), int(m.group(2)), int(m.group(3))
        if (major, minor) < MIN_PYTHON_VERSION:
            continue

        selected[(major, minor, patch)] = tag

    return [f"python:{tag}" for _, tag in sorted(selected.items())]


def extract_alpine_versions(golang_images: Iterable[str]) -> list[str]:
    """从 golang:X.Y.Z-alpineA.B 列表中反向提取所有用到的 alpine:A.B。

    结果按 alpine 版本升序排序、去重，用于生成 alpine 基础镜像清单。
    golang:X.Y.Z 这类 Debian tag 不携带 alpine 版本，会被跳过。
    """
    # 用集合去重，元素为 (alpine_major, alpine_minor)
    versions: set[tuple[int, int]] = set()
    for image in golang_images:
        # image 形如 "golang:1.25.6-alpine3.23"，去掉 "golang:" 前缀再匹配
        _, _, tag = image.partition(":")
        m = TAG_PATTERN.match(tag)
        if not m or m.group(4) is None or m.group(5) is None:
            continue
        versions.add((int(m.group(4)), int(m.group(5))))

    return [f"alpine:{major}.{minor}" for major, minor in sorted(versions)]


def render_output(
    golang_images: list[str],
    alpine_images: list[str],
    python_images: list[str],
    source_desc: str,
) -> str:
    """按 images.txt 的既有格式渲染输出内容。

    使用注释头 + 分节形式，便于人类阅读，也和现有 sync-runtime-images.py 的解析规则兼容
    （# 开头为注释，空行忽略，其余按 <name>:<tag> 处理）。
    """
    lines: list[str] = [
        "# 需要同步的镜像清单",
        "# 规则:",
        "#   1. 每行一个镜像, 格式为 <name>:<tag>",
        "#   2. 以 # 开头的行为注释, 空行会被忽略",
        "#   3. 行首/行尾空白会被自动裁剪",
        "#",
        f"# 本文件由 scripts/platform-build/gen-runtime-images.py 自动生成，数据源：{source_desc}",
        "# 请勿手工编辑，如需调整规则请修改脚本后重新生成",
        "",
        "# ========== golang 系列 ==========",
    ]
    lines.extend(golang_images)
    lines.append("")
    lines.append("# ========== alpine 系列 ==========")
    lines.extend(alpine_images)
    lines.append("")
    lines.append("# ========== python 系列 ==========")
    lines.extend(python_images)
    lines.append("")  # 末尾留一个空行，保持 POSIX 风格
    return "\n".join(lines)


def write_output(path: str, content: str) -> None:
    """将渲染好的内容写入目标文件，父目录不存在则直接报错。

    这里刻意不自动创建目录：configs/ 目录属于项目结构的一部分，缺失通常意味着
    用户搞错了工作目录，此时应该显式失败而不是悄悄创建。
    """
    parent = os.path.dirname(os.path.abspath(path))
    if not os.path.isdir(parent):
        raise FileNotFoundError(
            f"输出文件的父目录不存在：{parent}，请先确认目录已创建或路径是否正确"
        )
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="过滤 golang / python 镜像 tag，生成用于镜像同步的清单文件",
    )
    parser.add_argument(
        "--source",
        required=True,
        choices=["skopeo", "file"],
        help="tag 数据源：skopeo 表示直接调用命令拉取；file 表示从本地文件读取",
    )
    parser.add_argument(
        "--golang-input",
        help="当 --source=file 时必填，指定本地 golang tag 清单文件路径",
    )
    parser.add_argument(
        "--python-input",
        help="当 --source=file 时必填，指定本地 python tag 清单文件路径",
    )
    parser.add_argument(
        "--output",
        default=DEFAULT_OUTPUT_PATH,
        help=f"输出的镜像清单文件路径（默认：{DEFAULT_OUTPUT_PATH}）",
    )
    args = parser.parse_args()

    # file 模式下两份输入都必填，argparse 的 choices 无法直接表达联动关系，手动校验；
    # 缺 python 输入时不能静默跳过，否则重新生成会把已有的 python 系列整段删掉
    if args.source == "file" and not (args.golang_input and args.python_input):
        parser.error("--source=file 时必须同时提供 --golang-input <path> 与 --python-input <path>")
    return args


def main() -> int:
    args = parse_args()

    if args.source == "file":
        tags = read_tags_from_file(args.golang_input)
        python_tags = read_tags_from_file(args.python_input)
    else:
        tags = read_tags_from_skopeo()
        python_tags = read_tags_from_skopeo("docker://python")
    # 生成产物注释里只标注来源类型，避免把本地路径等易变信息写进文件
    source_desc = args.source

    golang_images = filter_and_pick_latest(tags)
    alpine_images = extract_alpine_versions(golang_images)
    python_images = filter_python_tags(python_tags)

    content = render_output(golang_images, alpine_images, python_images, source_desc)
    write_output(args.output, content)

    # 只打印一个简短摘要到 stdout，方便人类确认；详细内容看输出文件
    print(
        f"[OK] 写入 {args.output}："
        f"golang {len(golang_images)} 条, alpine {len(alpine_images)} 条, "
        f"python {len(python_images)} 条"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
