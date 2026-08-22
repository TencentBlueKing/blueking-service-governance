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

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const NAME = "bkms-cli";

function packageJSONPath() {
  if (process.env.npm_package_json && fs.existsSync(process.env.npm_package_json)) {
    return process.env.npm_package_json;
  }
  return path.join(__dirname, "..", "package.json");
}

function binaryPath() {
  const ext = process.platform === "win32" ? ".exe" : "";
  return path.join(__dirname, "..", "bin", NAME + ext);
}

function applyEndpoints() {
  const pkg = JSON.parse(fs.readFileSync(packageJSONPath(), "utf8"));
  const endpoints = pkg.bkmsCli || {};
  const bkmsBaseUrl = String(endpoints.bkmsBaseUrl || "").trim();
  const bcsApiHost = String(endpoints.bcsApiHost || "").trim();
  if (!bkmsBaseUrl && !bcsApiHost) {
    return;
  }

  const bin = binaryPath();
  if (!fs.existsSync(bin)) {
    throw new Error(`bkms-cli binary not found at ${bin}; install the binary first`);
  }

  const args = ["config", "set", "--if-unset"];
  if (bkmsBaseUrl) {
    args.push("--bkms-base-url", bkmsBaseUrl);
  }
  if (bcsApiHost) {
    args.push("--bcs-api-host", bcsApiHost);
  }
  execFileSync(bin, args, { stdio: "inherit" });
}

module.exports = { applyEndpoints, packageJSONPath, binaryPath };

if (require.main === module) {
  try {
    applyEndpoints();
  } catch (err) {
    console.error(`Failed to apply bkmsCli endpoints:`, err.message);
    process.exit(1);
  }
}
