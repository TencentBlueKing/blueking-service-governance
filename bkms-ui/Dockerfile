FROM node:23-alpine as build-stage

ARG MODE=production
ARG BKMS_APP_VERSION
ARG BK_CI_GIT_REPO_HEAD_COMMIT_ID
ARG BK_CI_BUILD_NUM

ENV BKMS_APP_VERSION=${BKMS_APP_VERSION}
ENV BK_CI_GIT_REPO_HEAD_COMMIT_ID=${BK_CI_GIT_REPO_HEAD_COMMIT_ID}
ENV BK_CI_BUILD_NUM=${BK_CI_BUILD_NUM}

WORKDIR /app
RUN corepack enable

COPY .npmrc package.json pnpm-lock.yaml ./
COPY patches/ ./patches/
RUN --mount=type=cache,id=pnpm-store,target=/root/.pnpm-store \
    pnpm install --frozen-lockfile

COPY . .
RUN pnpm build --mode ${MODE}

EXPOSE 5000

CMD ["pnpm", "run", "server"]
