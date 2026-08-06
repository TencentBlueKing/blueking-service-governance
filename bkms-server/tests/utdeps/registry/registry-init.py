#!/usr/bin/env python3

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

"""本脚本主要由 LLM 编写，用于初始化一个本地的 Docker Registry 服务，向其推送一个示例镜像。

成功执行后，registry 中将出现名为 REPOSITORY 的仓库，包含 TAGS 中指定的标签。REPOSITORY 和
TAGS 可通过环境变量设置。
"""

import hashlib
import json
import os
import sys
import time
import urllib.parse
import urllib.request
from contextlib import closing
from urllib.error import HTTPError, URLError

REGISTRY_URL = os.environ.get("REGISTRY_URL", "http://registry:5000").rstrip("/")
REPOSITORY = os.environ.get("REGISTRY_REPO", "fixture/sample").strip("/")
TAGS = [tag.strip() for tag in os.environ.get("REGISTRY_TAGS", "latest").split(",") if tag.strip()]

MANIFEST_MEDIA_TYPE = "application/vnd.docker.distribution.manifest.v2+json"
CONFIG_MEDIA_TYPE = "application/vnd.docker.container.image.v1+json"
LAYER_MEDIA_TYPE = "application/vnd.docker.image.rootfs.diff.tar"

LAYER_BYTES = b"registry fixture layer\n"
LAYER_DIFF_ID = "sha256:" + hashlib.sha256(LAYER_BYTES).hexdigest()

CONFIG_DOC = {
    "created": "2024-01-01T00:00:00Z",
    "architecture": "amd64",
    "os": "linux",
    "config": {},
    "rootfs": {"type": "layers", "diff_ids": [LAYER_DIFF_ID]},
    "history": [{"created": "2024-01-01T00:00:00Z", "created_by": "registry fixture init"}],
}
CONFIG_BYTES = json.dumps(CONFIG_DOC, separators=(",", ":")).encode("utf-8")


def log(message: str) -> None:
    print(message, flush=True)


def request(url: str, *, method: str = "GET", data: bytes | None = None, headers: dict[str, str] | None = None):
    all_headers = {"User-Agent": "registry-fixture/1.0"}
    if headers:
        all_headers.update(headers)
    req = urllib.request.Request(url, data=data, method=method, headers=all_headers)
    return urllib.request.urlopen(req, timeout=10)


def wait_for_registry(timeout_seconds: int = 60) -> None:
    deadline = time.time() + timeout_seconds
    ping_url = f"{REGISTRY_URL}/v2/_catalog"
    while time.time() < deadline:
        try:
            with closing(request(ping_url)) as resp:
                if resp.status < 400:
                    return
        except Exception:
            time.sleep(1)
    raise RuntimeError("timed out waiting for registry to become ready")


def repository_has_tags(expected: list[str]) -> bool:
    tags_url = f"{REGISTRY_URL}/v2/{REPOSITORY}/tags/list"
    try:
        with closing(request(tags_url)) as resp:
            payload = json.loads(resp.read().decode("utf-8"))
    except HTTPError as err:
        if err.code in (404, 400):
            return False
        raise
    except URLError:
        return False
    existing = set(payload.get("tags") or [])
    return all(tag in existing for tag in expected)


def start_blob_upload() -> str:
    start_url = f"{REGISTRY_URL}/v2/{REPOSITORY}/blobs/uploads/"
    with closing(request(start_url, method="POST", data=b"", headers={"Content-Length": "0"})) as resp:
        location = resp.headers.get("Location")
        if not location:
            raise RuntimeError("registry did not return upload location")
        return resolve_location(location)


def resolve_location(location: str) -> str:
    if location.startswith(("http://", "https://")):
        return location
    return urllib.parse.urljoin(f"{REGISTRY_URL}/", location.lstrip("/"))


def blob_exists(digest: str) -> bool:
    blob_url = f"{REGISTRY_URL}/v2/{REPOSITORY}/blobs/{digest}"
    try:
        with closing(request(blob_url, method="HEAD")):
            return True
    except HTTPError as err:
        if err.code == 404:
            return False
        raise


def upload_blob(payload: bytes) -> tuple[str, int]:
    digest = "sha256:" + hashlib.sha256(payload).hexdigest()
    size = len(payload)
    if blob_exists(digest):
        log(f"blob {digest} already exists")
        return digest, size
    upload_url = start_blob_upload()
    # Properly append digest parameter to URL that may already have query params
    separator = "&" if "?" in upload_url else "?"
    put_url = f"{upload_url}{separator}digest={urllib.parse.quote(digest, safe=':')}"
    with closing(request(put_url, method="PUT", data=payload, headers={"Content-Type": "application/octet-stream"})):
        log(f"blob {digest} uploaded")
    return digest, size


def push_manifest(manifest_bytes: bytes, tag: str) -> None:
    manifest_url = f"{REGISTRY_URL}/v2/{REPOSITORY}/manifests/{tag}"
    with closing(
        request(
            manifest_url,
            method="PUT",
            data=manifest_bytes,
            headers={"Content-Type": MANIFEST_MEDIA_TYPE},
        )
    ):
        log(f"tag {tag} published")


def main() -> None:
    if not TAGS:
        log("no tags configured, nothing to seed")
        return
    wait_for_registry()
    if repository_has_tags(TAGS):
        log("repository already seeded; exiting")
        return
    config_digest, config_size = upload_blob(CONFIG_BYTES)
    layer_digest, layer_size = upload_blob(LAYER_BYTES)
    manifest_doc = {
        "schemaVersion": 2,
        "mediaType": MANIFEST_MEDIA_TYPE,
        "config": {
            "mediaType": CONFIG_MEDIA_TYPE,
            "size": config_size,
            "digest": config_digest,
        },
        "layers": [
            {
                "mediaType": LAYER_MEDIA_TYPE,
                "size": layer_size,
                "digest": layer_digest,
            }
        ],
    }
    manifest_bytes = json.dumps(manifest_doc, separators=(",", ":")).encode("utf-8")
    for tag in TAGS:
        push_manifest(manifest_bytes, tag)
    log("registry fixture ready")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:  # noqa: BLE001 - surface full error for easier debugging
        log(f"registry init failed: {exc}")
        sys.exit(1)
