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
 * 第 3 步：检查。只读，不修改数据。
 *   1. 剩余 trpc / taf 应用逐个列出冲突原因，应与迁移的冲突清单一致；
 *   2. 指定 BATCH_ID 时，本批次已迁移的应用：type、framework、language 与迁移结果一致，
 *      AppModel 唯一且 frameworkConfigVersion=1、frameworkConfig 为空，框架 def 只有一条；
 *   3. 全部 bkmsApp 应用：技术栈在框架目录内，AppModel 唯一，框架 def 数量符合框架（trpc / taf 为 1，blank 为 0）。
 * 全部通过时输出 PASS，否则输出 FAIL 及明细。
 *
 * 用法：
 *   mongosh "<连接串>" --quiet --eval 'var BATCH_ID="20261110-1"' scripts/bkmsapp-migration/verify.js
 */

/* global load, __dirname, print, BKMS */
load(__dirname + "/lib.js");

(function () {
  BKMS.header("检查", "只读");
  var problems = [];

  // 1. 剩余旧类型应用
  var remaining = BKMS.legacyApps().map(function (app) {
    var p = BKMS.plan(app);
    return { app: app, reasons: p.conflicts.length > 0 ? p.conflicts : ["无冲突但未迁移"] };
  });
  remaining.forEach(function (r) {
    if (r.reasons[0] === "无冲突但未迁移") {
      problems.push(r.app.id + "：无冲突但仍是 " + r.app.type);
    }
  });

  // 2. 本批次迁移结果
  var batchChecked = 0;
  if (BKMS.cfg.batchID) {
    BKMS.db[BKMS.COLL.backups].find({ batchID: BKMS.cfg.batchID, migratedTo: { $ne: null } }).forEach(function (b) {
      batchChecked += 1;
      var app = BKMS.db[BKMS.COLL.apps].findOne({ id: b.appID });
      if (!app) {
        problems.push(b.appID + "：应用不存在");
        return;
      }
      if (app.type !== "bkmsApp" || app.framework !== b.migratedTo.framework || app.language !== b.migratedTo.language) {
        problems.push(b.appID + "：技术栈与迁移结果不一致（" + app.type + " / " + app.framework + " / " +
          app.language + "）");
      }
      var models = BKMS.modelsOf(b.appID);
      if (models.length === 1) {
        var w = models[0].workload || {};
        if (w.frameworkConfigVersion !== 1) {
          problems.push(b.appID + "：frameworkConfigVersion 为 " + w.frameworkConfigVersion);
        }
        if (BKMS.hasKeys(w.frameworkConfig)) {
          problems.push(b.appID + "：frameworkConfig 非空");
        }
      }
    });
  }

  // 3. 全部 bkmsApp 应用
  var total = 0;
  BKMS.db[BKMS.COLL.apps].find({ type: "bkmsApp" }).forEach(function (app) {
    total += 1;
    var langs = BKMS.CATALOG[app.framework];
    if (!langs) {
      problems.push(app.id + "：未知框架 " + app.framework);
      return;
    }
    if (langs.indexOf(app.language) < 0) {
      problems.push(app.id + "：目录外语言 " + app.framework + " + " + app.language);
    }
    var models = BKMS.modelsOf(app.id);
    if (models.length !== 1) {
      problems.push(app.id + "：AppModel 数量为 " + models.length);
    }
    var expected = BKMS.FRAMEWORK_FILE[app.framework] ? 1 : 0;
    var defs = BKMS.frameworkDefCount(app.id);
    if (defs !== expected) {
      problems.push(app.id + "：框架 def 数量为 " + defs + "，应为 " + expected);
    }
  });

  print("");
  print("bkmsApp 应用: " + total + "    剩余 trpc / taf 应用: " + remaining.length +
    (BKMS.cfg.batchID ? "    本批次已迁移: " + batchChecked : ""));
  print("");
  BKMS.printRows("[剩余旧类型应用]", remaining, function (r) {
    return r.app.workspaceID + " / " + r.app.id + "  " + r.app.type + "  " + r.reasons.join("；");
  });
  BKMS.printRows("[问题]", problems, function (r) {
    return r;
  });
  print("");
  print(problems.length === 0 ? "结论: PASS" : "结论: FAIL");
})();
