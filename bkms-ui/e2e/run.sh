#!/usr/bin/env bash
# ====================================================================
# BKMS E2E BDD 测试 Runner
# ====================================================================
# 本脚本串联"bddgen 生成 → 执行测试 → 生成 HTML 报告 → 上传 BKRepo"四个步骤。
#
# 用法:
#   cd apps/ui/e2e && ./run.sh [--profile smoke|deploy|config|readonly|all|spec] [--test-pattern <glob>] [--tags <expression>]
#
# 必需环境变量（与 e2e/.env.test 保持一致，无默认值）:
#   BKMS_TEST_ACCESS_TOKEN     蓝鲸测试环境 AccessToken
#   BKMS_TEST_SITE             目标站点
#   BKMS_TEST_DEFAULT_SPACE    测试空间
#   BKMS_TEST_REPORT_DIR       报告输出目录
#
# 可选环境变量:
#   BKMS_TEST_DEFAULT_ENV      默认测试环境
#   BKMS_TEST_DEFAULT_APP      默认测试应用（可空；缺省时按 appType 自动发现 e2e- 前缀应用）
#   BKMS_TEST_TRPC_APP         tRPC 测试应用覆盖值（推荐 e2e-trpc）
#   BKMS_TEST_HELM_APP         Helm 测试应用覆盖值（推荐 e2e-helm）
#   BK_CI                      CI 模式开关
#   BKMS_TEST_USE_MOCK         是否启用 mock 数据
#   TEST_NAME                  测试名称（用于目录命名）
#   BKREPO_USERNAME            BKRepo 用户名（可选，提供后自动上传）
#   BKREPO_TOKEN               BKRepo API Token
#   BKREPO_ADDR                BKRepo API 地址
#   BKREPO_PROJECT             BKRepo 项目名
#   BKREPO_REPO                BKRepo 仓库名
#   E2E_REPORT_FONT_BASE_URL   HTML 报告字体 CDN 根地址（可选）
#   E2E_PARALLEL               是否允许并行执行（默认 false；readonly profile 默认 true）
#   E2E_WORKERS                并发 worker 数（readonly profile 默认 2）
#   E2E_SKIP_PREFLIGHT         是否跳过目标站点预检
#   E2E_PREFLIGHT_TIMEOUT      目标站点预检超时时间，默认 10 秒
# ====================================================================
set -euo pipefail

# e2e/ 位于 apps/ui 下，SCRIPT_DIR = <repo>/apps/ui/e2e
# PROJECT_ROOT 指向仓库根目录（bkms-govern/），保证 BKMS_TEST_REPORT_DIR 等
# 相对路径被解析到仓库根，而非 apps/ui。
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# ── 解析参数 ──
TEST_PATTERN=""
TAGS_EXPR=""
PROFILE="all"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile) PROFILE="$2"; shift 2 ;;
    --test-pattern) TEST_PATTERN="$2"; shift 2 ;;
    --tags) TAGS_EXPR="$2"; shift 2 ;;
    *) echo "未知参数: $1" >&2; exit 1 ;;
  esac
done

case "$PROFILE" in
  smoke) PROFILE_TAGS="@smoke" ;;
  deploy) PROFILE_TAGS="@deploy-flow" ;;
  config) PROFILE_TAGS="@config-flow" ;;
  readonly) PROFILE_TAGS="@readonly" ;;
  all) PROFILE_TAGS="" ;;
  spec) PROFILE_TAGS="" ;;
  *) echo "未知 profile: $PROFILE，可选值: smoke, deploy, config, readonly, all, spec" >&2; exit 1 ;;
esac

# --tags 优先级高于 --profile
if [[ -z "$TAGS_EXPR" && -n "$PROFILE_TAGS" ]]; then
  TAGS_EXPR="$PROFILE_TAGS"
fi

if [[ "$PROFILE" == "readonly" ]]; then
  export E2E_PARALLEL="${E2E_PARALLEL:-true}"
  export E2E_WORKERS="${E2E_WORKERS:-2}"
fi

# ── 检查必需环境变量 ──
# 必填：access token / site / space / report dir（脚本运行所必需）
# 选填：env / app（仅当场景实际用到 @env:/@app: 时才需要配置，允许为空）
MISSING_VARS=()
if [[ -z "${BKMS_TEST_ACCESS_TOKEN:-}" ]]; then
  MISSING_VARS+=("BKMS_TEST_ACCESS_TOKEN")
fi
if [[ -z "${BKMS_TEST_SITE:-}" ]]; then
  MISSING_VARS+=("BKMS_TEST_SITE")
fi
if [[ -z "${BKMS_TEST_DEFAULT_SPACE:-}" ]]; then
  MISSING_VARS+=("BKMS_TEST_DEFAULT_SPACE")
fi
if [[ -z "${BKMS_TEST_REPORT_DIR:-}" ]]; then
  MISSING_VARS+=("BKMS_TEST_REPORT_DIR")
fi
if [[ ${#MISSING_VARS[@]} -gt 0 ]]; then
  echo "错误: 以下必需环境变量未设置: ${MISSING_VARS[*]}" >&2
  echo "请在 e2e/.env.test 中配置，或通过环境变量传入。" >&2
  exit 1
fi

# env / app 允许为空：显式导出空串，避免 dotenv 未加载时继承宿主机的同名变量
export BKMS_TEST_DEFAULT_ENV="${BKMS_TEST_DEFAULT_ENV:-}"
export BKMS_TEST_DEFAULT_APP="${BKMS_TEST_DEFAULT_APP:-}"
export BKMS_TEST_TRPC_APP="${BKMS_TEST_TRPC_APP:-}"
export BKMS_TEST_HELM_APP="${BKMS_TEST_HELM_APP:-}"

# ── 导出环境变量 ──
export BKMS_TEST_ACCESS_TOKEN
export BKMS_TEST_SITE
export BKMS_TEST_DEFAULT_SPACE
export BKMS_TEST_REPORT_DIR

TEST_NAME="${TEST_NAME:-e2e-bdd-test}"
TIMESTAMP="$(date +%Y%m%d-%H%M)"

# 若 BKMS_TEST_REPORT_DIR 为相对路径，解析为相对于项目根目录的绝对路径
if [[ "$BKMS_TEST_REPORT_DIR" != /* ]]; then
  BKMS_TEST_REPORT_DIR="$PROJECT_ROOT/$BKMS_TEST_REPORT_DIR"
  export BKMS_TEST_REPORT_DIR
fi
mkdir -p "$BKMS_TEST_REPORT_DIR"

echo "========================================"
echo " BKMS E2E BDD Test Runner"
echo "========================================"
echo " Site:       $BKMS_TEST_SITE"
echo " Space:      $BKMS_TEST_DEFAULT_SPACE"
echo " Env:        $BKMS_TEST_DEFAULT_ENV"
echo " App:        $BKMS_TEST_DEFAULT_APP"
echo " TRPC App:   $BKMS_TEST_TRPC_APP"
echo " Helm App:   $BKMS_TEST_HELM_APP"
echo " Report Dir: $BKMS_TEST_REPORT_DIR"
echo " Profile:    $PROFILE"
echo " Tags:       ${TAGS_EXPR:-<all>}"
echo " Parallel:   ${E2E_PARALLEL:-false}"
echo " Workers:    ${E2E_WORKERS:-1}"
echo "========================================"

if [[ "${E2E_SKIP_PREFLIGHT:-false}" != "true" ]]; then
  if [[ ! "$BKMS_TEST_SITE" =~ ^https?:// ]]; then
    echo "错误: BKMS_TEST_SITE 必须是 http(s) URL: $BKMS_TEST_SITE" >&2
    exit 1
  fi
  if command -v curl >/dev/null 2>&1; then
    PREFLIGHT_TIMEOUT="${E2E_PREFLIGHT_TIMEOUT:-10}"
    if ! curl --silent --show-error --location --max-time "$PREFLIGHT_TIMEOUT" --output /dev/null "$BKMS_TEST_SITE"; then
      echo "错误: 目标站点不可达: $BKMS_TEST_SITE" >&2
      echo "如需跳过站点预检，请设置 E2E_SKIP_PREFLIGHT=true。" >&2
      exit 1
    fi
  fi
fi

# ── Step 1: 安装依赖（仅首次） ──
# 使用 pnpm（与 Dockerfile 预装的 corepack pnpm@9.8.0 对齐），
# 并锁定 frozen-lockfile，确保镜像/宿主机/CI 三处依赖一致。
cd "$SCRIPT_DIR"
if [[ ! -d node_modules ]]; then
  echo ""
  echo "[Step 1/5] 安装 Node.js 依赖..."
  pnpm install --frozen-lockfile --silent
else
  echo ""
  echo "[Step 1/5] 依赖已安装，跳过。"
fi

if ! npx playwright --version >/dev/null 2>&1; then
  echo "错误: Playwright CLI 不可用，请检查依赖安装。" >&2
  exit 1
fi

# 确保 Chromium 可用
if ! npx playwright install --dry-run chromium >/dev/null 2>&1; then
  echo "安装 Chromium 浏览器..."
  npx playwright install chromium
fi

echo ""
echo "清理旧报告产物..."
rm -f "$BKMS_TEST_REPORT_DIR/result.json" "$BKMS_TEST_REPORT_DIR/report.html"
rm -f "$BKMS_TEST_REPORT_DIR"/*.png 2>/dev/null || true
rm -rf "$BKMS_TEST_REPORT_DIR/playwright-html"

# ── Step 2: bddgen 生成测试文件 ──
echo ""
if [[ "$PROFILE" == "spec" ]]; then
  echo "[Step 2/5] spec profile 不需要 bddgen，跳过。"
else
  echo "[Step 2/5] 从 .feature 文件生成 Playwright 测试代码..."
  if [[ -n "$TAGS_EXPR" ]]; then
    npx bddgen test --tags "$TAGS_EXPR"
  else
    npx bddgen test
  fi
  echo "测试文件已生成到 .features-gen/ 目录"
fi

# ── Step 3: 执行 Playwright 测试 ──
echo ""
echo "[Step 3/5] 执行 Playwright 测试..."
PLAYWRIGHT_EXIT=0
if [[ -n "$TEST_PATTERN" ]]; then
  if [[ "$PROFILE" == "spec" ]]; then
    npx playwright test --config=playwright.spec.config.ts "$TEST_PATTERN" || PLAYWRIGHT_EXIT=$?
  else
    npx playwright test "$TEST_PATTERN" || PLAYWRIGHT_EXIT=$?
  fi
else
  if [[ "$PROFILE" == "spec" ]]; then
    npx playwright test --config=playwright.spec.config.ts || PLAYWRIGHT_EXIT=$?
  else
    npx playwright test || PLAYWRIGHT_EXIT=$?
  fi
fi

if [[ $PLAYWRIGHT_EXIT -ne 0 ]]; then
  echo ""
  echo "⚠ Playwright 测试存在失败用例 (exit code: $PLAYWRIGHT_EXIT)"
  echo "  继续生成报告..."
fi

# ── Step 4: 生成 HTML 报告 ──
echo ""
echo "[Step 4/5] 生成 HTML 报告..."
RESULT_JSON="$BKMS_TEST_REPORT_DIR/result.json"
REPORT_HTML="$BKMS_TEST_REPORT_DIR/report.html"

if [[ -f "$RESULT_JSON" ]]; then
  python3 "$SCRIPT_DIR/scripts/generate_e2e_report.py" \
    --input "$RESULT_JSON" \
    --output "$REPORT_HTML" \
    --screenshots-dir "$BKMS_TEST_REPORT_DIR/"
  echo "HTML 报告: $REPORT_HTML"
else
  echo "警告: result.json 未找到，跳过 HTML 报告生成。" >&2
  echo "  预期路径: $RESULT_JSON"
fi

# ── Step 5: 上传到 BKRepo（可选） ──
BKREPO_TOKEN="${BKREPO_TOKEN:-}"
BKREPO_USERNAME="${BKREPO_USERNAME:-}"
BKREPO_ADDR="${BKREPO_ADDR:-}"
BKREPO_PROJECT="${BKREPO_PROJECT:-}"
BKREPO_REPO="${BKREPO_REPO:-}"

if [[ -n "$BKREPO_TOKEN" && -f "$REPORT_HTML" ]]; then
  echo ""
  echo "[Step 5/5] 上传报告到 BKRepo..."

  if [[ -z "$BKREPO_ADDR" || -z "$BKREPO_PROJECT" || -z "$BKREPO_REPO" ]]; then
    echo "警告: BKREPO_ADDR / BKREPO_PROJECT / BKREPO_REPO 未全部设置，跳过上传。" >&2
    BKREPO_URL=""
  else
    REPORT_FILENAME="bkms-e2e-bdd-${TIMESTAMP}-${TEST_NAME}.html"

    UPLOAD_ARGS=(
      --file "$REPORT_HTML"
      --api-addr "$BKREPO_ADDR"
      --api-token "$BKREPO_TOKEN"
      --project "$BKREPO_PROJECT"
      --repo "$BKREPO_REPO"
      --report-name "$REPORT_FILENAME"
    )
    if [[ -n "$BKREPO_USERNAME" ]]; then
      UPLOAD_ARGS+=(--username "$BKREPO_USERNAME")
    fi

    UPLOAD_OUTPUT=$(python3 "$SCRIPT_DIR/scripts/upload_to_bkrepo.py" "${UPLOAD_ARGS[@]}")
    echo "$UPLOAD_OUTPUT"
    # 提取上传后的文件 URL
    BKREPO_URL=$(echo "$UPLOAD_OUTPUT" | grep -oP '(?<=文件 URL: )\S+' || true)
  fi
else
  echo ""
  echo "[Step 5/5] BKRepo 凭证未提供或报告不存在，跳过上传。"
  BKREPO_URL=""
fi

# ── 统计测试结果 ──
TOTAL_CASES=0
PASSED_CASES=0
FAILED_CASES=0
if [[ -f "$RESULT_JSON" ]]; then
  eval "$(python3 -c "
import json, sys
with open('$RESULT_JSON') as f:
    data = json.load(f)
summary = data.get('summary', [])
total = len(summary)
passed = sum(1 for s in summary if s.get('status') == 'pass')
failed = total - passed
print(f'TOTAL_CASES={total}')
print(f'PASSED_CASES={passed}')
print(f'FAILED_CASES={failed}')
" 2>/dev/null || echo "")"
fi

# ── 设置蓝盾流水线变量（供后续步骤引用） ──
echo "::set-variable name=BKREPO_REPORT_URL::${BKREPO_URL:-}"
echo "::set-variable name=E2E_REPORT_DIR::${BKMS_TEST_REPORT_DIR}"
echo "::set-variable name=E2E_EXIT_CODE::${PLAYWRIGHT_EXIT}"
echo "::set-variable name=E2E_TOTAL_CASES::${TOTAL_CASES}"
echo "::set-variable name=E2E_PASSED_CASES::${PASSED_CASES}"
echo "::set-variable name=E2E_FAILED_CASES::${FAILED_CASES}"

# ── 输出摘要 ──
echo ""
echo "========================================"
echo " 执行完成"
echo "========================================"
echo " 测试结果: 共 ${TOTAL_CASES} 个用例，通过 ${PASSED_CASES} 个，失败 ${FAILED_CASES} 个"
echo " 报告目录: $BKMS_TEST_REPORT_DIR"
if [[ -f "$REPORT_HTML" ]]; then
  echo " HTML 报告: $REPORT_HTML"
fi
if [[ -f "$RESULT_JSON" ]]; then
  echo " JSON 结果: $RESULT_JSON"
fi
if [[ -n "$BKREPO_URL" ]]; then
  echo " BKRepo:    ${BKREPO_URL}?preview=true"
fi
echo "========================================"

exit $PLAYWRIGHT_EXIT
