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

"""
upload_to_bkrepo.py - 上传文件到 BKRepo

用法:
    # 推荐：通过环境变量提供地址/项目/仓库（见 e2e/.env.test）
    export BKREPO_ADDR=... BKREPO_PROJECT=... BKREPO_REPO=...
    python upload_to_bkrepo.py --file report.html --username <USER> --api-token <TOKEN>

    # 或通过 CLI 显式传入
    python upload_to_bkrepo.py --file report.html --username <USER> --api-token <TOKEN> \
        --api-addr <ADDR> --project <PROJECT> --repo <REPO> --report-name custom-name.html

输出:
    上传成功后打印文件访问 URL
"""

import argparse
import base64
import hashlib
import json
import os
import random
import string
import sys
import urllib.parse
import urllib.request
import urllib.error
from datetime import datetime


DEFAULT_EXPIRES = "7"  # 7 天过期


def gen_random_str(length=8):
    """生成随机字符串"""
    chars = string.ascii_letters + string.digits
    return "".join(random.choice(chars) for _ in range(length))


def upload_to_bkrepo(file_path, api_addr, api_token, project, repo, report_name="", username=None):
    """
    上传文件到 BKRepo，返回文件访问 URL。

    Args:
        file_path: 本地文件路径
        api_addr: BKRepo API 地址（来自 --api-addr / $BKREPO_ADDR）
        api_token: BKRepo API Token（密码/密钥）
        project: BKRepo 项目名（来自 --project / $BKREPO_PROJECT）
        repo: BKRepo 仓库名（来自 --repo / $BKREPO_REPO）
        report_name: 自定义文件名（可选）
        username: BKRepo 用户名（提供后自动生成 base64(username:token) 凭证）

    Returns:
        tuple: (file_url, response_data)
    """
    api_addr = api_addr.rstrip("/")

    # 若提供了 username，自动拼接 base64(username:token) 作为 Basic auth 凭证
    if username:
        api_token = base64.b64encode(f"{username}:{api_token}".encode()).decode()

    if report_name:
        filename = report_name
    else:
        now_str = datetime.now().strftime("%Y%m%d-%H%M%S")
        rand_str = gen_random_str(8)
        base_name = os.path.basename(file_path)
        filename = f"{now_str}-{rand_str}-{base_name}"

    upload_url = f"{api_addr}/generic/{project}/{repo}/{urllib.parse.quote(filename, safe='')}"

    with open(file_path, "rb") as f:
        file_data = f.read()

    sha256 = hashlib.sha256(file_data).hexdigest()
    md5 = hashlib.md5(file_data).hexdigest()

    headers = {
        "Authorization": f"Basic {api_token}",
        "Content-Type": "application/octet-stream",
        "X-BKREPO-SHA256": sha256,
        "X-BKREPO-MD5": md5,
        "X-BKREPO-OVERWRITE": "true",
        "X-BKREPO-EXPIRES": DEFAULT_EXPIRES,
    }

    req = urllib.request.Request(upload_url, data=file_data, headers=headers, method="PUT")

    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            resp_body = resp.read().decode("utf-8")
            resp_data = json.loads(resp_body)
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")
        print(f"上传失败, HTTP {e.code}: {body}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"上传失败: {e}", file=sys.stderr)
        sys.exit(1)

    if resp_data.get("code") != 0:
        print(f"上传失败, 接口返回: {json.dumps(resp_data, ensure_ascii=False)}", file=sys.stderr)
        sys.exit(1)

    full_path = resp_data.get("data", {}).get("fullPath", f"/{filename}")
    encoded_path = urllib.parse.quote(full_path, safe="/")
    file_url = f"{api_addr}/generic/{project}/{repo}{encoded_path}"

    return file_url, resp_data


def main():
    parser = argparse.ArgumentParser(description="上传文件到 BKRepo")
    parser.add_argument("--file", required=True, help="要上传的本地文件路径")
    parser.add_argument(
        "--api-addr",
        default=os.environ.get("BKREPO_ADDR"),
        help="BKRepo API 地址（默认读取 $BKREPO_ADDR）",
    )
    parser.add_argument("--username", default=os.environ.get("BKREPO_USERNAME"), help="BKRepo 用户名（默认读取 $BKREPO_USERNAME）")
    parser.add_argument("--api-token", default=os.environ.get("BKREPO_TOKEN"), help="BKRepo API Token（默认读取 $BKREPO_TOKEN）")
    parser.add_argument(
        "--project",
        default=os.environ.get("BKREPO_PROJECT"),
        help="BKRepo 项目名（默认读取 $BKREPO_PROJECT）",
    )
    parser.add_argument(
        "--repo",
        default=os.environ.get("BKREPO_REPO"),
        help="BKRepo 仓库名（默认读取 $BKREPO_REPO）",
    )
    parser.add_argument("--report-name", default="", help="自定义上传文件名（可选，不含路径）")

    args = parser.parse_args()

    missing = []
    if not args.api_token:
        missing.append("BKREPO_TOKEN / --api-token")
    if not args.api_addr:
        missing.append("BKREPO_ADDR / --api-addr")
    if not args.project:
        missing.append("BKREPO_PROJECT / --project")
    if not args.repo:
        missing.append("BKREPO_REPO / --repo")
    if missing:
        print(f"错误: 以下配置未提供: {', '.join(missing)}", file=sys.stderr)
        print("请在 e2e/.env.test 中配置，或通过环境变量 / CLI 参数传入。", file=sys.stderr)
        sys.exit(1)

    if not os.path.exists(args.file):
        print(f"文件不存在: {args.file}", file=sys.stderr)
        sys.exit(1)

    file_url, _ = upload_to_bkrepo(
        file_path=args.file,
        api_addr=args.api_addr,
        api_token=args.api_token,
        project=args.project,
        repo=args.repo,
        report_name=args.report_name,
        username=args.username,
    )

    print(f"上传成功!")
    print(f"文件 URL: {file_url}")


if __name__ == "__main__":
    main()
