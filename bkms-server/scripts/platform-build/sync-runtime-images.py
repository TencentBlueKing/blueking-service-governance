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
sync-runtime-images.py

用途：
    将镜像清单文件里的镜像从 Docker Hub 批量同步到指定的目标仓库
    （pull → tag → push）。用于在内部镜像仓库中就绪"平台构建"所需的
    运行时基础镜像。

依赖：
    - 本机已安装 docker 命令
    - 已 `docker login` 目标仓库

行为要点：
    - 目标仓库前缀末尾的多余 / 会被自动裁剪，兼容调用方拼写
    - 目标镜像已存在时主动跳过（通过 docker manifest inspect 探测）
    - 单个镜像同步失败不影响后续镜像，最终统一汇总"成功 / 跳过 / 失败"
    - golang 镜像强制要求 X.Y.Z 或 X.Y.Z-alpineA.B 精确小版本，防止误同步滚动 tag
    - 支持 --dry-run 只打印命令、不实际执行

清单文件格式：
    - 每行一个镜像，形如 golang:1.24.13 或 golang:1.24.13-alpine3.23
    - 以 # 开头的行为注释，空行会被忽略
    - 行内 # 之后的部分会被当作注释裁剪掉，首尾空白同样裁剪

使用示例：
    # 正式同步
    uv run scripts/platform-build/sync-runtime-images.py \
        --target-registry mirrors.tencent.com/bkms \
        --input configs/workload-runtime-images.txt

    # 只预览命令，不真正执行
    uv run scripts/platform-build/sync-runtime-images.py \
        --target-registry mirrors.tencent.com/bkms \
        --input configs/workload-runtime-images.txt \
        --dry-run
"""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from enum import Enum

# 校验 golang 镜像 tag 必须是 X.Y.Z 或 X.Y.Z-alpineA.B 形式的精确小版本
# 避免同步 golang:1.24、golang:1.24-alpine 之类的滚动 tag 造成不确定性
GOLANG_TAG_PATTERN = re.compile(r"^golang:\d+\.\d+\.\d+(?:-alpine\d+\.\d+)?$")


class SyncResult(Enum):
    """单个镜像同步结果，用于 main 中统一分类汇总。"""

    SUCCESS = "success"
    SKIPPED = "skipped"  # 目标镜像已存在，主动跳过
    FAILED = "failed"  # pull/tag/push 过程出错


def normalize_registry(registry: str) -> str:
    """裁剪目标仓库前缀末尾多余的 /，并校验非空。

    兼容调用方传入 "mirrors.tencent.com/bkms" 或 "mirrors.tencent.com/bkms/"
    这两种写法。
    """
    normalized = registry.rstrip("/")
    if not normalized:
        raise ValueError("目标仓库前缀不能为空")
    return normalized


def load_images(path: str) -> list[str]:
    """从清单文件读取镜像列表。

    过滤规则：
    - 去掉行内 # 之后的注释
    - 去掉首尾空白
    - 过滤掉空行
    """
    if not os.path.isfile(path):
        raise FileNotFoundError(f"配置文件不存在: {path}")

    images: list[str] = []
    with open(path, "r", encoding="utf-8") as f:
        for raw_line in f:
            # 先去掉行内注释，再裁剪首尾空白
            line = raw_line.split("#", 1)[0].strip()
            if line:
                images.append(line)
    return images


def validate_image(image: str) -> None:
    """校验镜像配置，避免同步不符合平台规则的 tag。

    目前只针对 golang 镜像做严格校验：必须是 X.Y.Z 或 X.Y.Z-alpineA.B 精确小版本。
    其他镜像不作限制，交由 docker 自身报错。
    """
    if image.startswith("golang:") and not GOLANG_TAG_PATTERN.match(image):
        raise ValueError(f"golang 镜像必须使用 X.Y.Z 或 X.Y.Z-alpineA.B 小版本: {image}")


def run_command(cmd: list[str], *, dry_run: bool) -> None:
    """执行外部命令，dry_run 模式下只打印不执行。

    统一带上 "+ " 前缀打印，观感与 bash 的 `set -x` 类似，便于人肉核对
    实际下发的命令。命令失败时抛出 CalledProcessError，由上层捕获归类。
    """
    # 简单地用空格拼接展示，命令行元素本身不含空格时足够阅读
    print("+ " + " ".join(cmd))
    if dry_run:
        return
    subprocess.run(cmd, check=True)


def image_exists_remote(image: str) -> bool:
    """通过 docker manifest inspect 探测目标镜像是否已经存在于远端仓库。

    manifest inspect 只查询 manifest 元信息，不会拉取镜像层，开销很小。
    stderr 一并丢弃，避免 "manifest unknown" 之类的报错在正常流程里刷屏。
    """
    result = subprocess.run(
        ["docker", "manifest", "inspect", image],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    return result.returncode == 0


def sync_image(src: str, target_registry: str, *, dry_run: bool) -> SyncResult:
    """同步单个镜像：pull → tag → push。

    - 先做规则校验，不符合直接判失败
    - 再探测目标镜像是否已存在，存在则跳过整个流程
    - 最后按顺序执行 pull / tag / push，任一步失败即判定该镜像失败
    """
    dst = f"{target_registry}/{src}"

    try:
        validate_image(src)
    except ValueError as e:
        print(f"错误: {e}", file=sys.stderr)
        return SyncResult.FAILED

    print(f"=== 同步 {src} -> {dst} ===")

    # 优先检查目标镜像是否已存在，存在则跳过，避免重复拉取/推送
    if image_exists_remote(dst):
        print(f"目标镜像已存在, 跳过: {dst}")
        return SyncResult.SKIPPED

    try:
        run_command(["docker", "pull", src], dry_run=dry_run)
        run_command(["docker", "tag", src, dst], dry_run=dry_run)
        run_command(["docker", "push", dst], dry_run=dry_run)
    except subprocess.CalledProcessError as e:
        print(f"错误: docker 命令执行失败 (exit={e.returncode}): {' '.join(e.cmd)}", file=sys.stderr)
        return SyncResult.FAILED

    return SyncResult.SUCCESS


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="将镜像清单里的镜像从 Docker Hub 批量同步到指定目标仓库",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=(
            "示例:\n"
            "  uv run scripts/platform-build/sync-runtime-images.py \\\n"
            "      --target-registry mirrors.tencent.com/bkms \\\n"
            "      --input configs/workload-runtime-images.txt\n"
        ),
    )
    parser.add_argument(
        "--target-registry",
        required=True,
        help="目标仓库前缀，例如 mirrors.tencent.com/bkms，末尾多余的 / 会被自动裁剪",
    )
    parser.add_argument(
        "--input",
        required=True,
        help="镜像清单文件路径，例如 configs/workload-runtime-images.txt",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="只打印将要执行的 docker 命令，不真正执行",
    )
    return parser.parse_args()


def print_summary(
    total: int,
    synced: int,
    skipped: list[str],
    failed: list[str],
) -> None:
    """汇总打印同步结果，格式与旧 shell 版本保持一致，便于用户目视对比。"""
    print()
    print("===== 同步结果汇总 =====")
    print(f"总计:   {total}")
    print(f"成功:   {synced}")
    print(f"跳过:   {len(skipped)}")
    print(f"失败:   {len(failed)}")

    if skipped:
        print()
        print("以下镜像因目标已存在被跳过:")
        for image in skipped:
            print(f"  - {image}")

    if failed:
        print()
        print("以下镜像同步失败:")
        for image in failed:
            print(f"  - {image}")


def main() -> int:
    args = parse_args()

    try:
        target_registry = normalize_registry(args.target_registry)
    except ValueError as e:
        print(f"错误: {e}", file=sys.stderr)
        return 1

    # 是否只预览命令、不实际执行
    dry_run = args.dry_run

    print(f"目标仓库前缀: {target_registry}")
    print(f"使用配置文件: {args.input}")
    if dry_run:
        print("(dry-run 模式：只打印命令，不实际执行)")

    try:
        images = load_images(args.input)
    except FileNotFoundError as e:
        print(f"错误: {e}", file=sys.stderr)
        return 1

    if not images:
        print("配置文件中没有可同步的镜像")
        return 0

    skipped: list[str] = []
    failed: list[str] = []
    synced = 0
    for image in images:
        result = sync_image(image, target_registry, dry_run=dry_run)
        if result is SyncResult.SUCCESS:
            synced += 1
        elif result is SyncResult.SKIPPED:
            skipped.append(image)
        else:
            failed.append(image)

    print_summary(len(images), synced, skipped, failed)

    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
