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

/*
 * 测试数据：在独立的测试库中构造各类样本，用于本地验证迁移脚本。库名必须以 _fixture 结尾，执行时会先清空样本集合。
 *
 * 用法：
 *   mongosh "<连接串>" --quiet --eval 'var DB_NAME="bkms_migration_fixture"' scripts/bkmsapp-migration/fixtures.js
 *
 * 期望结果（migrate.js）：可迁移 3 个（trpc-go、trpc-cpp、taf），冲突 7 个，bkmsApp 与 helm 应用不处理。
 */

/* global load, __dirname, print, BKMS */
load(__dirname + "/lib.js");

(function () {
  if (!/_fixture$/.test(BKMS.db.getName())) {
    throw new Error("fixtures.js only runs against a database whose name ends with _fixture");
  }
  [BKMS.COLL.apps, BKMS.COLL.models, BKMS.COLL.defs, BKMS.COLL.backups].forEach(function (c) {
    BKMS.db[c].deleteMany({});
  });

  var tenantID = "default";
  function app(id, fields) {
    BKMS.db[BKMS.COLL.apps].insertOne(Object.assign(
      { id: id, name: id, displayName: id, workspaceID: "ws-fixture", tenantID: tenantID }, fields
    ));
  }
  function model(appID, workload) {
    BKMS.db[BKMS.COLL.models].insertOne({
      appID: appID, tenantID: tenantID, workload: Object.assign({ name: appID }, workload),
    });
  }
  function defs(appID, n) {
    for (var i = 0; i < n; i++) {
      BKMS.db[BKMS.COLL.defs].insertOne({
        appID: appID, tenantID: tenantID, name: "conf-" + i, configKind: "framework", mountDir: "/conf",
      });
    }
  }

  // 可迁移
  app("ok-trpc-go", { type: "trpc", trpcSpec: { language: "go" } });
  model("ok-trpc-go", { type: "trpc", trpcConfig: { language: "go" } });
  defs("ok-trpc-go", 1);

  app("ok-trpc-cpp", { type: "trpc", trpcSpec: { language: "cpp" } });
  model("ok-trpc-cpp", { type: "trpc" });
  defs("ok-trpc-cpp", 1);

  app("ok-taf", { type: "taf" });
  model("ok-taf", { type: "taf", tafConfig: { fileContent: "<taf/>" } });
  defs("ok-taf", 1);

  // 冲突
  app("bad-lang-conflict", { type: "trpc", trpcSpec: { language: "go" } });
  model("bad-lang-conflict", { type: "trpc", trpcConfig: { language: "cpp" } });
  defs("bad-lang-conflict", 1);

  app("bad-lang-missing", { type: "trpc" });
  model("bad-lang-missing", { type: "trpc" });
  defs("bad-lang-missing", 1);

  app("bad-lang-unknown", { type: "trpc", trpcSpec: { language: "java" } });
  model("bad-lang-unknown", { type: "trpc" });
  defs("bad-lang-unknown", 1);

  app("bad-type-conflict", { type: "trpc", trpcSpec: { language: "go" } });
  model("bad-type-conflict", { type: "taf" });
  defs("bad-type-conflict", 1);

  app("bad-dual-config", { type: "trpc", trpcSpec: { language: "go" } });
  model("bad-dual-config", { type: "trpc", trpcConfig: { language: "go" }, tafConfig: { fileName: "a.conf" } });
  defs("bad-dual-config", 1);

  app("bad-orphan", { type: "trpc", trpcSpec: { language: "go" } });
  defs("bad-orphan", 1);

  app("bad-two-defs", { type: "taf" });
  model("bad-two-defs", { type: "taf" });
  defs("bad-two-defs", 2);

  // 不处理
  app("new-blank-go", { type: "bkmsApp", framework: "blank", language: "go" });
  model("new-blank-go", { frameworkConfigVersion: 1 });

  app("helm-app", { type: "helm" });

  print("fixtures created in " + BKMS.db.getName() + ": " + BKMS.db[BKMS.COLL.apps].countDocuments({}) + " apps");
})();
