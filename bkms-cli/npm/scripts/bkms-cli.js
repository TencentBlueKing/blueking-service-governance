#!/usr/bin/env node
/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

// 这个文件是 npm 装好后，用户敲 `bkms-cli` 时真正跑到的入口（package.json 的 bin）。
//
// 它自己不实现 CLI 逻辑，只是找到同级目录下 postinstall 下载好的原生二进制，
// 把命令行参数原样转过去。如果二进制不存在（比如 postinstall 被跳过或下载失败），
// 会打印怎么手动重跑 install.js。
//
// Windows 上如果之前用过自更新，有可能留下一个改名失败的 .old 文件；这里会尽量
// 把它恢复成可用的 bkms-cli，避免命令直接挂掉。

const { execFileSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const ext = process.platform === "win32" ? ".exe" : "";
const bin = path.join(__dirname, "..", "bin", "bkms-cli" + ext);

// On Windows, a crashed self-update may have left the binary renamed to .old.
// Recover it before proceeding so the CLI remains functional.
const oldBin = bin + ".old";
function restoreOldBinary() {
  try {
    if (fs.existsSync(bin)) {
      fs.rmSync(bin, { force: true });
    }
    fs.renameSync(oldBin, bin);
    return true;
  } catch (_) {
    return false;
  }
}

if (process.platform === "win32" && fs.existsSync(oldBin)) {
  if (!fs.existsSync(bin)) {
    restoreOldBinary();
  } else {
    try {
      execFileSync(bin, ["--version"], { stdio: "ignore", timeout: 10000 });
      try {
        fs.rmSync(oldBin, { force: true });
      } catch (_) {
        // Best-effort cleanup; keep running the healthy binary.
      }
    } catch (_) {
      restoreOldBinary();
    }
  }
}

if (!fs.existsSync(bin)) {
  console.error(
    `Error: bkms-cli binary not found at ${bin}\n\n` +
      `This usually means the postinstall script was skipped.\n` +
      `Common causes:\n` +
      `  - npm is configured with ignore-scripts=true\n` +
      `  - The postinstall download failed\n\n` +
      `To fix, run the install script manually:\n` +
      `  node "${path.join(__dirname, "install.js")}"\n`
  );
  process.exit(1);
}

try {
  execFileSync(bin, process.argv.slice(2), {
    stdio: "inherit",
    env: { ...process.env, BKMS_CLI_INSTALL_SOURCE: "npm" },
  });
} catch (e) {
  process.exit(e.status || 1);
}
