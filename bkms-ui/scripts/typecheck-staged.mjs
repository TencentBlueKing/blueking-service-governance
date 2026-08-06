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

// pre-commit 类型检查：仅检查暂存区 TypeScript/Vue 文件，禁止 explicit any
import { spawn, spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const uiRoot = path.resolve(scriptDir, '..');
const cacheDir = path.join(uiRoot, 'node_modules', '.cache', 'typecheck-staged');

// 仅检查 src/ 下源文件，排除自动生成的 API 类型
const SOURCE_PREFIX = 'src/';
const EXCLUDED_SOURCE_PREFIX = 'src/api/modules/';
const TYPE_SCRIPT_FILE_PATTERN = /\.(?:ts|tsx|vue)$/;
// 适配 vue-tsc / tsc 不同版本的报错输出格式
const DIAGNOSTIC_PATH_PATTERNS = [/^(.+?)\(\d+,\d+\):\s+error\s+TS\d+:/, /^(.+):\d+:\d+\s+-\s+error\s+TS\d+:/];

// 统一路径分隔符
const normalizeSlashes = value => value.replace(/\\/g, '/');

// 同步执行命令
const run = (command, args, options = {}) =>
  spawnSync(command, args, {
    encoding: 'utf8',
    shell: process.platform === 'win32',
    ...options,
  });

// 异步执行命令，返回 { kill, promise } 支持并行和取消
const runAsync = (command, args, options = {}) => {
  const child = spawn(command, args, {
    cwd: options.cwd,
    windowsHide: true,
  });
  const chunks = { stderr: [], stdout: [] };

  child.stdout.on('data', chunk => chunks.stdout.push(chunk));
  child.stderr.on('data', chunk => chunks.stderr.push(chunk));

  return {
    kill: () => child.kill(),
    promise: new Promise((resolve, reject) => {
      child.on('error', reject);
      child.on('close', (status, signal) => {
        resolve({
          signal,
          status,
          stderr: Buffer.concat(chunks.stderr).toString('utf8'),
          stdout: Buffer.concat(chunks.stdout).toString('utf8'),
        });
      });
    }),
  };
};

// 命令失败时输出错误并退出
const exitWithCommandFailure = (result, message) => {
  process.stderr.write(result.stderr || result.stdout);
  process.stderr.write(`\n[typecheck:staged] ${message}\n`);
  process.exit(result.status || 1);
};

// 获取 Git 仓库根目录
const getRepoRoot = () => {
  const result = run('git', ['rev-parse', '--show-toplevel'], { cwd: uiRoot });

  if (result.status !== 0) {
    exitWithCommandFailure(result, 'Unable to locate git repository.');
  }

  return result.stdout.trim();
};

// 获取暂存区新增/修改文件列表（排除删除）
const getStagedFiles = repoRoot => {
  const result = run('git', ['-C', repoRoot, 'diff', '--cached', '--name-only', '--diff-filter=ACMR']);

  if (result.status !== 0) {
    exitWithCommandFailure(result, 'Unable to inspect staged files.');
  }

  return result.stdout
    .split(/\r?\n/)
    .map(file => normalizeSlashes(file.trim()))
    .filter(Boolean);
};

// 筛选暂存区中需检查的 TypeScript/Vue 文件（去重，排除 API 生成文件）
const getStagedUiTypeFiles = (stagedFiles, uiPrefix) => [
  ...new Set(
    stagedFiles
      .filter(file => file.startsWith(uiPrefix) && TYPE_SCRIPT_FILE_PATTERN.test(file))
      .map(file => file.slice(uiPrefix.length))
      .filter(file => file.startsWith(SOURCE_PREFIX) && !file.startsWith(EXCLUDED_SOURCE_PREFIX)),
  ),
];

// 生成临时 tsconfig，继承项目配置但仅 include 暂存文件
const createTempTsconfig = stagedTypeFiles => ({
  extends: normalizeSlashes(path.relative(cacheDir, path.join(uiRoot, 'tsconfig.json'))),
  include: [
    '../../bkui-vue/lib/volar.components.d.ts', // 保留全局类型声明
    '../../../src/@types/**/*',
    '../../../src/shims.d.ts', // 引入全局类型声明
    '../../../src/types/**/*.d.ts',
    '../../../src/*.d.ts', // 保留 src/ 根目录下的全局声明文件（shims.d.ts, vite-env.d.ts 等）
    ...stagedTypeFiles.map(file => `../../../${file}`),
  ],
});

// 生成独立 ESLint 配置，仅启用 no-explicit-any 规则
const createExplicitAnyEslintConfig = () => `import tsParser from '@typescript-eslint/parser';
import tsPlugin from '@typescript-eslint/eslint-plugin';
import vueParser from 'vue-eslint-parser';

const explicitAnyRule = {
  plugins: { '@typescript-eslint': tsPlugin },
  rules: { '@typescript-eslint/no-explicit-any': 'error' },
};

export default [
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      parser: tsParser,
      parserOptions: { ecmaVersion: 'latest', jsx: true, sourceType: 'module' },
    },
    ...explicitAnyRule,
  },
  {
    files: ['**/*.vue'],
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        ecmaVersion: 'latest',
        extraFileExtensions: ['.vue'],
        parser: tsParser,
        sourceType: 'module',
      },
    },
    ...explicitAnyRule,
  },
];
`;

// 在缓存目录创建临时配置文件（文件名含 PID+时间戳，避免并发冲突）
const createTempFiles = stagedTypeFiles => {
  fs.mkdirSync(cacheDir, { recursive: true });

  const suffix = `${process.pid}.${Date.now()}`;
  const tsconfigPath = path.join(cacheDir, `tsconfig.${suffix}.json`);
  const eslintConfigPath = path.join(cacheDir, `eslint.${suffix}.mjs`);

  fs.writeFileSync(tsconfigPath, `${JSON.stringify(createTempTsconfig(stagedTypeFiles), null, 2)}\n`);
  fs.writeFileSync(eslintConfigPath, createExplicitAnyEslintConfig());

  return {
    eslintConfigPath,
    paths: [tsconfigPath, eslintConfigPath],
    tsconfigPath,
  };
};

// 清理临时文件（失败静默忽略）
const cleanupFiles = files => {
  for (const file of files) {
    try {
      fs.unlinkSync(file);
    } catch {
      // 尽力清理
    }
  }
};

// 输出子进程结果，成功写 stdout，失败写 stderr
const printCommandOutput = result => {
  const output = `${result.stdout || ''}${result.stderr || ''}`;

  if (output.trim()) {
    process[result.status === 0 ? 'stdout' : 'stderr'].write(output);
  }

  return output;
};

// 将诊断路径统一为相对于 uiRoot 的路径
const normalizeDiagnosticPath = (filePath, uiPrefix) => {
  const normalized = normalizeSlashes(filePath.trim());

  if (path.isAbsolute(normalized)) {
    return normalizeSlashes(path.relative(uiRoot, normalized));
  }

  if (normalized.startsWith(uiPrefix)) {
    return normalized.slice(uiPrefix.length);
  }

  return normalized.replace(/^\.\//, '');
};

// 从 vue-tsc 输出中筛选属于暂存文件的错误行
const findStagedDiagnostics = (output, stagedTypeFiles, uiPrefix) =>
  output.split(/\r?\n/).filter(line => {
    const match = DIAGNOSTIC_PATH_PATTERNS.map(pattern => line.match(pattern)).find(Boolean);

    return match && stagedTypeFiles.has(normalizeDiagnosticPath(match[1], uiPrefix));
  });

// 判断输出是否包含 TS 诊断错误
const hasTypeScriptDiagnostics = output =>
  output.split(/\r?\n/).some(line => DIAGNOSTIC_PATH_PATTERNS.some(pattern => pattern.test(line)));

// 并行执行 explicit any 检查和类型检查；若 any 不通过则提前终止类型检查
const runChecks = async (stagedTypeFiles, tempFiles) => {
  const nodeBin = process.execPath;
  const eslintBin = path.join(uiRoot, 'node_modules', 'eslint', 'bin', 'eslint.js');
  const vueTscBin = path.join(uiRoot, 'node_modules', 'vue-tsc', 'bin', 'vue-tsc.js');

  const explicitAnyProcess = runAsync(
    nodeBin,
    [eslintBin, '--no-config-lookup', '--config', tempFiles.eslintConfigPath, '--no-warn-ignored', ...stagedTypeFiles],
    { cwd: uiRoot },
  );
  const typecheckProcess = runAsync(
    nodeBin,
    [vueTscBin, '--noEmit', '--pretty', 'false', '-p', tempFiles.tsconfigPath],
    { cwd: uiRoot },
  );

  const explicitAnyResult = await explicitAnyProcess.promise;
  printCommandOutput(explicitAnyResult);

  if (explicitAnyResult.status !== 0) {
    typecheckProcess.kill();
    await typecheckProcess.promise;
    process.stderr.write('[typecheck:staged] Explicit any is not allowed in staged TypeScript/Vue file(s).\n');
    return { status: explicitAnyResult.status || 1 };
  }

  return { typecheckResult: await typecheckProcess.promise };
};

// 主流程：获取暂存文件 → 生成临时配置 → 并行检查 → 按结果阻断或放行
const main = async () => {
  const repoRoot = getRepoRoot();
  const uiPrefix = `${normalizeSlashes(path.relative(repoRoot, uiRoot))}/`;
  const stagedTypeFiles = getStagedUiTypeFiles(getStagedFiles(repoRoot), uiPrefix);

  if (stagedTypeFiles.length === 0) {
    console.log(
      '[typecheck:staged] No staged .ts/.tsx/.vue files under apps/ui/src, excluding src/api/modules. Skipping typecheck.',
    );
    return 0;
  }

  console.log(`[typecheck:staged] Checking ${stagedTypeFiles.length} staged .ts/.tsx/.vue file(s) under apps/ui/src.`);

  const tempFiles = createTempFiles(stagedTypeFiles);

  try {
    const checkResult = await runChecks(stagedTypeFiles, tempFiles);

    if (checkResult.status) {
      return checkResult.status;
    }

    const { typecheckResult } = checkResult;
    const output = `${typecheckResult.stdout || ''}${typecheckResult.stderr || ''}`;

    // 类型检查通过
    if (typecheckResult.status === 0) {
      if (output.trim()) {
        process.stdout.write(output);
      }
      return 0;
    }

    const stagedTypeFileSet = new Set(stagedTypeFiles);
    const matchedDiagnostics = findStagedDiagnostics(output, stagedTypeFileSet, uiPrefix);

    // 暂存文件中有类型错误 → 阻断提交
    if (matchedDiagnostics.length > 0) {
      process.stderr.write('[typecheck:staged] TypeScript typecheck failed for staged file(s):\n\n');
      process.stderr.write(`${matchedDiagnostics.join('\n')}\n\n`);
      process.stderr.write('[typecheck:staged] Fix the staged TypeScript/Vue type errors before committing.\n');
      return typecheckResult.status || 1;
    }

    // 错误均在非暂存文件 → 放行
    if (hasTypeScriptDiagnostics(output)) {
      console.log(
        '[typecheck:staged] Typecheck reported errors, but none are in staged TypeScript/Vue files. Allowing commit.',
      );
      return 0;
    }

    // 无法解析的输出（兜底）→ 阻断
    process.stderr.write(output);
    process.stderr.write(
      '\n[typecheck:staged] Typecheck failed without parseable TypeScript diagnostics. Blocking commit.\n',
    );
    return typecheckResult.status || 1;
  } finally {
    cleanupFiles(tempFiles.paths);
  }
};

process.exit(await main());
