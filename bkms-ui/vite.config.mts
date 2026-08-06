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

// import viteCompression from 'vite-plugin-compression';
import VueI18n from '@intlify/unplugin-vue-i18n/vite';
import Vue from '@vitejs/plugin-vue';
import VueJsx from '@vitejs/plugin-vue-jsx';
import figlet from 'figlet';
import path from 'node:path';
import Unocss from 'unocss/vite';
import Components from 'unplugin-vue-components/vite';
import { defineConfig, loadEnv } from 'vite';
import VueDevTools from 'vite-plugin-vue-devtools';
import Layouts from 'vite-plugin-vue-layouts';
import SvgLoader from 'vite-svg-loader';

export default ({ mode }: { mode: string }) => {
  const env = loadEnv(mode, process.cwd(), 'BK_');
  const isDev = mode === 'development';
  return defineConfig({
    base: env.BK_STATIC_URL,

    server: {
      host: env.BK_APP_HOST,
      port: Number(env.BK_APP_PORT),
      allowedHosts: [env.BK_ALLOWED_HOST],
      // https: {},
      open: true,
      proxy: {
        '/simple_account': {
          target: env.BK_API_BASE_URL,
          changeOrigin: true,
          secure: true,
        },
        '/bkms': {
          target: env.BK_API_BASE_URL,
          changeOrigin: true,
          secure: true,
          toProxy: true,
          headers: {
            referer: env.BK_API_BASE_URL,
            origin: env.BK_API_BASE_URL,
          },
        },
        '/generic': {
          target: env.BK_STATIC_URL,
          changeOrigin: true,
          secure: true,
        },
        '/bcsapi': {
          target: env.BK_BCS_API_BASE_URL,
          changeOrigin: true,
          secure: false, // 忽略证书错误
          toProxy: true,
          headers: {
            referer: env.BK_BCS_API_BASE_URL,
          },
        },
        '/ms': {
          target: env.BK_REPO_URL,
          changeOrigin: true,
          secure: true,
          toProxy: true,
          headers: {
            referer: env.BK_REPO_URL,
          },
        },
        '/api-bk-user-selector': {
          target: (env.BK_API_URL_TMPL || '').replace('{api_name}', 'bk-user-web'),
          changeOrigin: true,
          secure: false,
          cookieDomainRewrite: '',
          rewrite: path => 'prod/' + path.replace(/^\/api-bk-user-selector/, ''),
          headers: {
            'sec-fetch-site': 'same-origin',
            'sec-fetch-mode': 'cors',
            'sec-fetch-dest': 'empty',
          },
        },
      },
    },

    optimizeDeps: {
      include: ['monaco-editor/esm/vs/language/json/json.worker', 'monaco-editor/esm/vs/editor/editor.worker'],
    },

    // 环境变量前缀
    envPrefix: 'BK_',

    // 生产镜像不产出 .map，降低源码泄露风险（Nginx 侧仍会拒绝 .map 访问）
    build: {
      sourcemap: isDev,
    },

    // 路径别名
    resolve: {
      alias: {
        '~/': `${path.resolve(__dirname, 'src')}/`,
        '@/': `${path.resolve(__dirname, 'src')}/`,
      },
    },

    plugins: [
      // https://github.com/vitejs/vite-plugin-vue/tree/main/packages/plugin-vue
      Vue(),

      // viteCompression({
      //   filter: /\.js|.css$/,
      //   threshold: 1,
      // }),
      // https://github.com/vitejs/vite-plugin-vue/tree/main/packages/plugin-vue-jsx
      VueJsx(),

      // https://github.com/JohnCampionJr/vite-plugin-vue-layouts
      Layouts(),

      // https://github.com/antfu/unplugin-vue-components
      Components({
        // allow auto load markdown components under `./src/components/`
        extensions: ['vue', 'md'],
        // allow auto import and register components used in markdown
        include: [/\.vue$/, /\.vue\?vue/, /\.md$/],
        dts: 'src/components.d.ts',
      }),

      // https://github.com/antfu/unocss
      // see uno.config.ts for config
      Unocss({
        inspector: isDev, // 仅在开发环境启用 UnoCSS 检查器 https://unocss.net/tools/inspector
      }),

      // https://github.com/intlify/bundle-tools/tree/main/packages/unplugin-vue-i18n
      VueI18n({
        runtimeOnly: true,
        compositionOnly: true,
        fullInstall: true,
        include: [path.resolve(__dirname, 'locales/**')],
      }),

      // https://github.com/jpkleemans/vite-svg-loader
      SvgLoader({ defaultImport: 'component' }),

      // 仅本地 dev server 启用，避免生产/镜像构建占用额外内存
      ...(isDev ? [VueDevTools()] : []),

      // 注入版本号到 index.html
      {
        name: 'html-version',
        transformIndexHtml(html) {
          return html.replace('__BK_BKMS_APP_VERSION__', process.env.BKMS_APP_VERSION || '--');
        },
      },
    ],

    // https://github.com/vitest-dev/vitest
    test: {
      include: ['test/**/*.test.ts'],
      environment: 'jsdom',
    },

    ssr: {
      // TODO: workaround until they support native ESM
      noExternal: ['workbox-window', /vue-i18n/],
    },

    // 定义变量
    define: {
      // 欢迎语
      BK_BKMS_WELCOME: JSON.stringify(
        figlet.textSync('Welcome To BKMS', {
          width: 120,
        }),
      ),
      // 版本
      BK_BKMS_VERSION: JSON.stringify(
        `version: ${process.env.BKMS_APP_VERSION || '--'}, commitID: ${process.env.BK_CI_GIT_REPO_HEAD_COMMIT_ID || '--'}, build: ${process.env.BK_CI_BUILD_NUM || 'dev'}`,
      ),
    },
  });
};
