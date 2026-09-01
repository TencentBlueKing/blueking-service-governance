# ====================================================================
# BKMS E2E 测试运行环境 - 基础依赖镜像
# ====================================================================
# 本镜像仅包含运行 test/e2e/run.sh 所需的基础依赖：
#   - Node.js 22 + corepack/pnpm（与 apps/ui packageManager 一致，pnpm install / pnpm exec playwright）
#   - Chromium 浏览器 + 系统依赖库（Playwright 测试执行）
#   - Python 3（HTML 报告生成 + BKRepo 上传）
#   - Git（拉取项目代码）
#
# 不包含 e2e 测试代码，需在容器内通过 git clone 获取项目后运行。
#
# 重要：镜像中预装了 Playwright 1.59.1 的 Chromium 浏览器，
# 并通过 PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 跳过 pnpm install 时的浏览器下载，
# e2e/package.json 中 @playwright/test 必须精确锁定为 "1.59.1" 以保持版本一致。
# 升级 Playwright 版本时需同步修改此 Dockerfile 的预装版本号并重建镜像。
#
# 构建（在 apps/ui 目录）:
#   docker build -t bkms-e2e-runner:latest -f e2e/Dockerfile .
#
# 使用（环境变量与 e2e/.env.test 对齐；测试应用变量可空，缺省时自动发现 e2e- 前缀应用）:
#   docker run -it --rm \
#     -e BKMS_TEST_ACCESS_TOKEN="your-token" \
#     -e BKMS_TEST_SITE="https://bkms.example.com" \
#     -e BKMS_TEST_DEFAULT_SPACE="bkmsstage" \
#     -e BKMS_TEST_DEFAULT_ENV="" \
#     -e BKMS_TEST_DEFAULT_APP="" \
#     -e BKMS_TEST_HELM_APP="" \
#     -e BKMS_TEST_REPORT_DIR="test-reports/default" \
#     bkms-e2e-runner:latest bash
#
#   # 容器内
#   git clone <repo-url> project && cd project/apps/ui
#   pnpm install
#   ./e2e/run.sh --tags "@TC-01 or @TC-02 or @TC-03"
# ====================================================================

FROM node:22-slim

# 与 apps/ui package.json 的 packageManager 对齐，供 run.sh 使用 pnpm
RUN corepack enable && corepack prepare pnpm@9.8.0 --activate

# 流水线要求：bash 为默认 shell 解释器
SHELL ["/bin/bash", "-c"]

# 流水线要求：使用 root 用户
USER root

# 安装系统依赖：
#   - curl（流水线下载 agent）
#   - bash（默认 shell）
#   - default-jdk（流水线 agent 运行所需 JDK 环境）
#   - Playwright Chromium 运行所需的系统库
#   - python3（HTML 报告生成 + BKRepo 上传）
#   - git（拉取代码）
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    bash \
    git \
    python3 \
    python3-pip \
    ca-certificates \
    default-jre-headless \
    # Playwright Chromium 所需的系统库
    libnss3 \
    libnspr4 \
    libdbus-1-3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxkbcommon0 \
    libatspi2.0-0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libpango-1.0-0 \
    libcairo2 \
    libasound2 \
    libx11-xcb1 \
    fonts-liberation \
    fonts-noto-color-emoji \
    fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*

# 预装 Playwright 1.59.1 的 Chromium 浏览器（仅 Chromium，不装 Firefox/WebKit）
# 并跳过后续 pnpm install 时的浏览器下载，始终使用镜像中预装的版本
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
RUN npx playwright@1.59.1 install chromium

# 流水线要求：清除 ENTRYPOINT，使用 CMD /bin/sh 启动
ENTRYPOINT []
CMD ["/bin/sh"]

WORKDIR /workspace
