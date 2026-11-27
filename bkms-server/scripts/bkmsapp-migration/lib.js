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
 * bkmsApp 存量迁移的公共规则，由 backup / migrate / verify / restore / cleanup 通过 load() 引入。
 *
 * 可在 --eval 中预先设置：
 *   DB_NAME     库名，默认 bkms_server
 *   BATCH_ID    批次号，backup / migrate / restore 必填
 *   APPLY       为 true 时写库，否则只打印（migrate / restore / cleanup）
 *   MAX_DETAIL  明细最多打印条数，默认 200
 *
 * 规则说明见 design_notes/bkms_app_runtime_model_migration.md §4。
 */

/* global db, print, printjson */

var BKMS = (function () {
  var cfg = {
    dbName: typeof DB_NAME !== "undefined" ? DB_NAME : "bkms_server",
    batchID: typeof BATCH_ID !== "undefined" ? String(BATCH_ID) : "",
    apply: typeof APPLY !== "undefined" && (APPLY === true || APPLY === "true"),
    maxDetail: typeof MAX_DETAIL !== "undefined" ? MAX_DETAIL : 200,
  };

  // 与 pkg/core/appruntime/catalog.go 保持一致
  var CATALOG = { trpc: ["cpp", "go"], taf: ["cpp"], blank: ["cpp", "go", "nodejs", "python"] };
  // 支持框架配置文件的框架，与能力 frameworkFile 保持一致
  var FRAMEWORK_FILE = { trpc: true, taf: true, blank: false };
  var LEGACY_TYPES = ["trpc", "taf"];

  var COLL = {
    apps: "applications",
    models: "app_models",
    defs: "app_config_file_defs",
    backups: "bkmsapp_migration_backups",
  };

  var database = db.getSiblingDB(cfg.dbName);

  function nonEmpty(v) {
    return v !== undefined && v !== null && v !== "";
  }

  function uniq(values) {
    var out = [];
    values.forEach(function (v) {
      if (nonEmpty(v) && out.indexOf(v) < 0) {
        out.push(v);
      }
    });
    return out;
  }

  function hasKeys(obj) {
    return obj !== undefined && obj !== null && typeof obj === "object" && Object.keys(obj).length > 0;
  }

  // trpcConfig / tafConfig 中任一字段非空即视为存在
  function hasConfig(cfgObj) {
    if (!cfgObj) {
      return false;
    }
    return ["fileName", "filePath", "fileContent", "language"].some(function (k) {
      return nonEmpty(cfgObj[k]);
    });
  }

  function requireBatch() {
    if (!cfg.batchID) {
      throw new Error("BATCH_ID is required, e.g. --eval 'var BATCH_ID=\"20261110-1\"'");
    }
  }

  // 按 appID 读取 AppModel，返回数组以便识别重复记录
  function modelsOf(appID) {
    return database[COLL.models].find({ appID: appID }).toArray();
  }

  function frameworkDefCount(appID) {
    return database[COLL.defs].countDocuments({ appID: appID, configKind: "framework" });
  }

  /*
   * 判断一个旧类型应用能否迁移，返回 { framework, language, conflicts }。
   * conflicts 为空时可迁移。
   */
  function plan(app) {
    var conflicts = [];
    var framework = app.type;
    var models = modelsOf(app.id);
    var model = models.length === 1 ? models[0] : null;
    var workload = (model && model.workload) || {};

    if (models.length === 0) {
      conflicts.push("缺少 AppModel");
    } else if (models.length > 1) {
      conflicts.push("AppModel 重复：" + models.length + " 条");
    }

    // 语言：tRPC 取 trpcSpec / trpcConfig / 已有 language；TAF 固定为 cpp
    var specLang = app.trpcSpec ? app.trpcSpec.language : "";
    var configLang = workload.trpcConfig ? workload.trpcConfig.language : "";
    var candidates = uniq([app.language, specLang, configLang]);
    var language = "";
    if (framework === "trpc") {
      if (candidates.length === 0) {
        conflicts.push("语言缺失");
      } else if (candidates.length > 1) {
        conflicts.push("语言冲突：" + candidates.join(" / "));
      } else {
        language = candidates[0];
      }
    } else {
      language = "cpp";
      if (candidates.length > 0 && !(candidates.length === 1 && candidates[0] === "cpp")) {
        conflicts.push("TAF 语言不是 cpp：" + candidates.join(" / "));
      }
    }
    if (language && CATALOG[framework].indexOf(language) < 0) {
      conflicts.push("目录外语言：" + framework + " + " + language);
    }

    // workload.type 只比较产品框架值，与 appruntime 规范化规则一致
    if (nonEmpty(workload.type) && CATALOG[workload.type] && workload.type !== framework) {
      conflicts.push("类型冲突：type=" + app.type + "，workload.type=" + workload.type);
    }
    if (hasConfig(workload.trpcConfig) && hasConfig(workload.tafConfig)) {
      conflicts.push("trpcConfig 与 tafConfig 同时存在");
    }
    if (nonEmpty(app.framework) && app.framework !== framework) {
      conflicts.push("已有 framework 不一致：" + app.framework);
    }
    if (hasKeys(workload.frameworkConfig)) {
      conflicts.push("frameworkConfig 非空");
    }
    if (nonEmpty(workload.frameworkConfigVersion) && workload.frameworkConfigVersion !== 1) {
      conflicts.push("frameworkConfigVersion 不是 1：" + workload.frameworkConfigVersion);
    }
    var defCount = frameworkDefCount(app.id);
    if (defCount !== 1) {
      conflicts.push("框架 def 数量为 " + defCount + "（需先用 repair_trpc_taf_framework_defs 处理）");
    }
    return { framework: framework, language: language, conflicts: conflicts };
  }

  function legacyApps() {
    return database[COLL.apps].find({ type: { $in: LEGACY_TYPES } }).toArray();
  }

  function printRows(title, rows, fmt) {
    print(title + "（共 " + rows.length + " 条，最多显示 " + cfg.maxDetail + " 条）");
    rows.slice(0, cfg.maxDetail).forEach(function (r) {
      print("  " + fmt(r));
    });
  }

  // mode 为空时按 APPLY 显示写库或只读
  function header(name, mode) {
    print("==================== bkmsApp 迁移：" + name + " ====================");
    print(
      "库: " + database.getName() +
        "    批次: " + (cfg.batchID || "-") +
        "    模式: " + (mode || (cfg.apply ? "APPLY（写库）" : "DRY-RUN（只读）")) +
        "    时间: " + new Date().toISOString()
    );
  }

  return {
    cfg: cfg,
    db: database,
    COLL: COLL,
    CATALOG: CATALOG,
    FRAMEWORK_FILE: FRAMEWORK_FILE,
    LEGACY_TYPES: LEGACY_TYPES,
    nonEmpty: nonEmpty,
    hasKeys: hasKeys,
    requireBatch: requireBatch,
    modelsOf: modelsOf,
    frameworkDefCount: frameworkDefCount,
    plan: plan,
    legacyApps: legacyApps,
    printRows: printRows,
    header: header,
  };
})();
