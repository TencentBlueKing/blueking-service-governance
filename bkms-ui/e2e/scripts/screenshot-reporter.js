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

/**
 * 自定义 Playwright Reporter。
 *
 * 收集每个测试步骤的结果，在测试结束后输出与 bkms-e2e-test 兼容的 result.json。
 * 截图由测试代码自行保存到 BKMS_TEST_REPORT_DIR，本 reporter 通过以下两种方式收集截图信息：
 * 1. Playwright step 跟踪：在 onStepEnd 中检测截图步骤（步骤标题含"截图"关键词）
 * 2. 目录扫描：在 onEnd 中扫描 BKMS_TEST_REPORT_DIR 目录下的 .png 文件作为兜底
 */
const fs = require('fs');
const path = require('path');

class ScreenshotReporter {
  constructor(options = {}) {
    this.outputDir = options.outputDir || process.env.BKMS_TEST_REPORT_DIR;
    if (!this.outputDir) {
      throw new Error('未设置报告输出目录：请通过 reporter options.outputDir 或环境变量 BKMS_TEST_REPORT_DIR 指定');
    }
    this.steps = [];
    this.suiteResults = [];
    this.startTime = null;
    // 记录每个测试（Scenario）关联的截图名称
    // key: test.id, value: string[] (截图文件名列表)
    this._testScreenshots = new Map();
    this._currentTestId = null;
  }

  onBegin(_config, suite) {
    this.startTime = new Date();
    fs.mkdirSync(this.outputDir, { recursive: true });
  }

  onTestBegin(test) {
    this._currentTestId = test.id;
    if (!this._testScreenshots.has(test.id)) {
      this._testScreenshots.set(test.id, []);
    }
  }

  onStepBegin(_test, _result, step) {
    // noop
  }

  onStepEnd(test, _result, step) {
    // 检测截图步骤：标题通常类似 '截图 "01-app-list-loaded"' 或 'And 截图 "..."'
    const title = step.title || '';
    const screenshotMatch = title.match(/截图\s+"([^"]+)"/);
    if (screenshotMatch) {
      const screenshotName = screenshotMatch[1];
      const testId = test.id;
      if (!this._testScreenshots.has(testId)) {
        this._testScreenshots.set(testId, []);
      }
      this._testScreenshots.get(testId).push(`${screenshotName}.png`);
    }
  }

  onTestEnd(test, result) {
    const suiteName = test.parent ? test.parent.title : '';
    const caseIdMatch = suiteName.match(/TC-\d+/);
    const caseId = caseIdMatch ? caseIdMatch[0] : suiteName;
    const caseName = suiteName.replace(/^TC-\d+\s*/, '');

    // 获取此测试关联的截图列表
    const screenshots = this._testScreenshots.get(test.id) || [];

    // 如果通过 step 跟踪没有收集到截图，尝试从 Playwright attachments 获取
    if (screenshots.length === 0 && result.attachments) {
      for (const att of result.attachments) {
        if (att.contentType && att.contentType.startsWith('image/') && att.path) {
          screenshots.push(path.basename(att.path));
        }
      }
    }

    const annotations = test.annotations || [];
    const stepAnnotations = annotations.filter((a) => a.type === 'step_result');

    if (stepAnnotations.length > 0) {
      for (const ann of stepAnnotations) {
        try {
          const data = JSON.parse(ann.description);
          this.steps.push({
            case_id: caseId,
            case_name: caseName,
            step_num: data.step_num,
            title: data.title,
            command: data.command || '',
            result: data.result || '',
            assertion: data.assertion || '',
            status: data.status || (result.status === 'passed' ? 'pass' : 'fail'),
            screenshot: data.screenshot || '',
          });
        } catch {
          // skip malformed annotations
        }
      }
    } else {
      // 将截图列表合并为逗号分隔的字符串
      const screenshotStr = screenshots.join(',');

      this.steps.push({
        case_id: caseId,
        case_name: caseName,
        step_num: 1,
        title: test.title,
        command: '',
        result: result.status === 'passed' ? '通过' : (result.error?.message || '失败'),
        assertion: result.status === 'passed' ? '通过' : '失败',
        status: result.status === 'passed' ? 'pass' : 'fail',
        screenshot: screenshotStr,
      });
    }

    const existing = this.suiteResults.find((s) => s.caseId === caseId);
    if (existing) {
      existing.total += 1;
      if (result.status === 'passed') existing.passed += 1;
      else existing.failed += 1;
    } else {
      this.suiteResults.push({
        caseId,
        caseName,
        fullName: suiteName,
        total: 1,
        passed: result.status === 'passed' ? 1 : 0,
        failed: result.status !== 'passed' ? 1 : 0,
      });
    }
  }

  onEnd(result) {
    const env = process.env;
    const testTime = this.startTime
      ? this.startTime.toISOString().slice(0, 16).replace('T', ' ')
      : new Date().toISOString().slice(0, 16).replace('T', ' ');

    const caseNames = this.suiteResults.map((s) => s.fullName).join(' | ');

    const summary = this.suiteResults.map((s) => ({
      name: s.fullName,
      total_steps: s.total,
      passed: s.passed,
      failed: s.failed,
      status: s.failed === 0 ? 'pass' : 'fail',
    }));

    // 兜底：如果步骤中没有截图信息，扫描目录中的 .png 文件分配到步骤
    this._assignScreenshotsFromDirectory();

    const issues = this.steps
      .filter((s) => s.status === 'fail')
      .map((s) => ({
        case_id: s.case_id,
        step: `步骤${s.step_num}`,
        description: s.result,
        suggestion: '',
      }));

    const data = {
      meta: {
        test_time: testTime,
        test_site: env.BKMS_TEST_SITE || '',
        auth_method: 'AccessToken (Bearer)',
        test_cases: caseNames,
        space: env.BKMS_TEST_DEFAULT_SPACE || '',
        env: env.BKMS_TEST_DEFAULT_ENV || '',
        app: env.BKMS_TEST_DEFAULT_APP || '',
        report_dir: path.relative(process.cwd(), this.outputDir),
      },
      summary,
      steps: this.steps,
      issues,
      recommendations: issues.length === 0
        ? '所有测试步骤均通过，系统运行正常。'
        : `共 ${issues.length} 个步骤失败，请检查相关功能。`,
    };

    const outputPath = path.join(this.outputDir, 'result.json');
    fs.writeFileSync(outputPath, JSON.stringify(data, null, 2), 'utf-8');
  }

  /**
   * 兜底方案：扫描 BKMS_TEST_REPORT_DIR 中的 .png 截图文件，
   * 按文件名前缀（如 01-, 02-, 03-）的自然顺序分配到步骤上。
   * 仅当步骤尚无截图信息时生效。
   */
  _assignScreenshotsFromDirectory() {
    // 如果已经有截图信息则不覆盖
    const hasAnyScreenshots = this.steps.some((s) => s.screenshot);
    if (hasAnyScreenshots) return;

    try {
      const files = fs.readdirSync(this.outputDir);
      const pngFiles = files
        .filter((f) => f.endsWith('.png'))
        .sort(); // 按字母序排序（01-xxx, 02-xxx, 03-xxx...）

      if (pngFiles.length === 0) return;

      // 按步骤顺序分配截图
      if (pngFiles.length === this.steps.length) {
        // 截图数量与步骤数量一致，1:1 分配
        for (let i = 0; i < this.steps.length; i++) {
          this.steps[i].screenshot = pngFiles[i];
        }
      } else {
        // 数量不一致，尝试通过编号前缀匹配
        for (const png of pngFiles) {
          const numMatch = png.match(/^(\d+)-/);
          if (numMatch) {
            const idx = parseInt(numMatch[1], 10) - 1; // 01 → index 0
            if (idx >= 0 && idx < this.steps.length) {
              // 追加到已有的截图列表
              if (this.steps[idx].screenshot) {
                this.steps[idx].screenshot += `,${png}`;
              } else {
                this.steps[idx].screenshot = png;
              }
            }
          }
        }
      }
    } catch {
      // 目录扫描失败不影响主流程
    }
  }
}

module.exports = ScreenshotReporter;
