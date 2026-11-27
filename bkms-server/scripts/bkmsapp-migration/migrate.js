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
 * 第 2 步：迁移。把本批次已备份、无冲突的 trpc / taf 应用改为 bkmsApp：
 *   1. AppModel 写入 workload.frameworkConfigVersion=1（frameworkConfig 为空对象时不落库，与新建应用一致）；
 *   2. Application 写入 type=bkmsApp、framework、language，最后写，作为完成标记。
 * 两步都按备份时的原值做条件更新，可重复执行、可分批执行；旧字段全部保留。
 *
 * 用法（默认只打印，加 APPLY=true 写库）：
 *   mongosh "<连接串>" --quiet --eval 'var BATCH_ID="20261110-1"' scripts/bkmsapp-migration/migrate.js
 *   mongosh "<连接串>" --quiet --eval 'var BATCH_ID="20261110-1"; var APPLY=true' scripts/bkmsapp-migration/migrate.js
 */

/* global load, __dirname, print, BKMS */
load(__dirname + "/lib.js");

(function () {
  BKMS.requireBatch();
  BKMS.header("迁移");

  var backups = BKMS.db[BKMS.COLL.backups];
  var ready = [];
  var conflicts = [];
  var skipped = [];
  var migrated = [];

  // 当前值是否仍与备份一致，不一致说明备份后有修改，需要重新备份
  function sameAsBackup(app, b) {
    var specLang = app.trpcSpec ? app.trpcSpec.language || null : null;
    var bSpecLang = b.trpcSpec ? b.trpcSpec.language || null : null;
    return app.type === b.type &&
      (app.framework || null) === b.framework &&
      (app.language || null) === b.language &&
      specLang === bSpecLang;
  }

  BKMS.legacyApps().forEach(function (app) {
    var backup = backups.findOne({ batchID: BKMS.cfg.batchID, appID: app.id });
    if (!backup) {
      skipped.push({ app: app, reason: "未备份，请先执行 backup.js" });
      return;
    }
    if (!sameAsBackup(app, backup.application)) {
      skipped.push({ app: app, reason: "备份后已修改，请重新备份" });
      return;
    }
    var p = BKMS.plan(app);
    if (p.conflicts.length > 0) {
      conflicts.push({ app: app, reasons: p.conflicts });
      return;
    }
    ready.push({ app: app, plan: p, backup: backup });
  });

  if (BKMS.cfg.apply) {
    ready.forEach(function (item) {
      var app = item.app;
      var b = item.backup;
      BKMS.db[BKMS.COLL.models].updateOne(
        { appID: app.id, "workload.frameworkConfigVersion": b.appModel.frameworkConfigVersion },
        { $set: { "workload.frameworkConfigVersion": 1 } }
      );
      var res = BKMS.db[BKMS.COLL.apps].updateOne(
        {
          id: app.id,
          type: b.application.type,
          framework: b.application.framework,
          language: b.application.language,
          "trpcSpec.language": b.application.trpcSpec ? b.application.trpcSpec.language || null : null,
        },
        { $set: { type: "bkmsApp", framework: item.plan.framework, language: item.plan.language } }
      );
      if (res.matchedCount !== 1) {
        skipped.push({ app: app, reason: "写入时应用已被修改，未迁移" });
        return;
      }
      backups.updateOne(
        { _id: b._id },
        { $set: { migratedTo: { framework: item.plan.framework, language: item.plan.language, at: new Date() } } }
      );
      migrated.push(item);
    });
  }

  print("");
  print("旧类型应用: " + (ready.length + conflicts.length + skipped.length) +
    "    可迁移: " + ready.length + "    冲突: " + conflicts.length + "    跳过: " + skipped.length +
    (BKMS.cfg.apply ? "    已迁移: " + migrated.length : ""));
  print("");
  BKMS.printRows(BKMS.cfg.apply ? "[已迁移]" : "[可迁移]", BKMS.cfg.apply ? migrated : ready, function (r) {
    return r.app.workspaceID + " / " + r.app.id + "  " + r.app.type + " -> bkmsApp + " +
      r.plan.framework + " + " + r.plan.language;
  });
  BKMS.printRows("[冲突，需人工处理]", conflicts, function (r) {
    return r.app.workspaceID + " / " + r.app.id + "  " + r.app.type + "  " + r.reasons.join("；");
  });
  BKMS.printRows("[跳过]", skipped, function (r) {
    return r.app.workspaceID + " / " + r.app.id + "  " + r.reason;
  });
})();
