# bkms-ui 编码规范

## 1. License Header

**每个新增源文件必须在文件顶部包含项目 License header。**

### 1.1 规则

- **不要凭记忆编写**，从同类型现有文件中逐字复制
- TypeScript / JavaScript 的 header 由 `eslint.config.mjs` 中的 `codecc/license` 规则强制检查
- Vue SFC 与 CSS 当前不在该规则覆盖范围内，新增时必须按仓库全局 `AGENTS.md` 手工检查

### 1.2 按文件类型

| 文件类型                      | 格式                                        | 参考文件                  |
| ----------------------------- | ------------------------------------------- | ------------------------- |
| TypeScript / JavaScript / CSS | `/* ... */` 块注释，文件最顶部，后跟空行    | `src/main.ts`             |
| Vue SFC                       | `<!-- ... -->` HTML 注释，`<template>` 之前 | `src/App.vue`             |
| Python                        | `#` 行注释，在 shebang 之后                 | `bkms-server/scripts/...` |

### 1.3 例外（无需 header）

- 自动生成文件：
  - `src/@types/v1/**`（`pnpm gen:api:v1` 生成）
  - `src/components.d.ts`（`unplugin-vue-components` 生成）
  - Go `// Code generated ... DO NOT EDIT` 文件
- `src/fonts/iconcool.js`（压缩图标资源）

### 1.4 检查与修复

```bash
pnpm lint        # 检查 JS / TS 的 header，同时执行常规 ESLint 规则
pnpm lint:fix    # 可修复 JS / TS 的缺失 header
```

> `pnpm lint:fix` 不会为 Vue SFC 或 CSS 插入 header。这两类文件应从 `src/App.vue`、`src/main.ts` 等同类型参考文件逐字复制并人工复核。

## 2. 文件命名

### 2.1 通用规则

- 统一使用**连字符（kebab-case）**格式
- 示例：`user.ts`、`request-queue.ts`、`home-page.vue`、`deploy-history.vue`

### 2.2 各目录约定

| 目录           | 命名         | 示例                                              |
| -------------- | ------------ | ------------------------------------------------- |
| `pages/`       | 连字符       | `application.vue`、`cluster-health-diagnosis.vue` |
| `components/`  | 连字符       | `member-selector.vue`、`overflow-tags.vue`        |
| `composables/` | `use-xxx.ts` | `use-deploy-status.ts`、`use-table-sort.ts`       |
| `stores/`      | 连字符       | `app-detail.ts`、`deploy-env.ts`                  |
| `api/modules/` | 连字符       | `app-config-files.ts`、`helm-charts.ts`           |

## 3. 组件开发

### 3.1 组件放置

| 类型       | 位置                                           | 说明               |
| ---------- | ---------------------------------------------- | ------------------ |
| 通用组件   | `src/components/`                              | 自动导入，全局可用 |
| 页面级组件 | `src/pages/{模块}/components/`                 | 仅当前模块使用     |
| 业务子组件 | `src/pages/{模块}/detail/{子模块}/components/` | 仅当前子模块使用   |

### 3.2 组件命名

- **文件名**：连字符 — `user-profile.vue`
- **组件名**（defineComponent name / script setup 自动推导）：大驼峰 — `UserProfile`
- **模板引用**：大驼峰 — `<UserProfile />`

### 3.3 自动导入

`src/components/` 下的组件由 `unplugin-vue-components` 自动注册：

```vue
<!-- 无需 import，直接使用 -->
<template>
  <FlexRow>
    <MonacoEditor :value="code" />
  </FlexRow>
</template>
```

> `src/components.d.ts` 自动生成类型声明，**勿手动编辑**。

### 3.4 组件结构（Vue 3 Composition API + script setup）

```vue
<!--
 - License header（从 App.vue 复制）
-->

<template>
  <div class="my-component">
    <!-- 模板内容 -->
  </div>
</template>

<script setup lang="ts">
  import { ref, computed } from 'vue';
  import { useMyStore } from '~/stores/my-feature';

  // Props
  const props = defineProps<{
    title: string;
    count?: number;
  }>();

  // Emits
  const emit = defineEmits<{
    change: [value: string];
    submit: [];
  }>();

  // 响应式状态
  const data = ref<string>('');

  // 计算属性
  const displayTitle = computed(() => props.title.toUpperCase());

  // 方法
  function handleSubmit() {
    emit('submit');
  }
</script>

<style scoped>
  .my-component {
    /* 样式 */
  }
</style>
```

## 4. 路由与布局

### 4.1 路由定义

路由在 `src/modules/router.ts` 中定义，使用 `setupLayouts` 包装：

```ts
{
  path: ':space/my-page',
  name: 'myPage',
  component: MyPage,
  meta: {
    layout: 'content',   // 指定布局
    menuId: 'MY_PAGE',   // 菜单标识
  },
}
```

### 4.2 布局类型

| 布局      | 文件                  | 用途                            |
| --------- | --------------------- | ------------------------------- |
| `main`    | `layouts/main.vue`    | 首页布局（RouterView + Footer） |
| `content` | `layouts/content.vue` | 侧边栏 + 内容区                 |
| `default` | `layouts/default.vue` | 默认布局                        |
| `empty`   | `layouts/empty.vue`   | 空白布局（仅 RouterView）       |

通过 `route.meta.layout` 指定，未指定则使用 `default`。

### 4.3 动态路由（CustomRouterComponent）

应用详情等页面使用 `CustomRouterComponent` 按 `menuId` + `menuName` 动态加载子页面：

1. 菜单配置在 `src/config/navigation/{模块}.ts`
2. 路由定义 `meta.menuId`
3. `CustomRouterComponent` 根据 `menuName` 匹配并渲染对应组件

## 5. 状态管理 (Pinia)

### 5.1 Store 定义

```ts
// src/stores/my-feature.ts
import { ref } from 'vue';
import { defineStore } from 'pinia';

export const useMyStore = defineStore('my-feature', () => {
  const data = ref<string>('');
  const loading = ref(false);

  function setData(value: string) {
    data.value = value;
  }

  return { data, loading, setData };
});
```

### 5.2 持久化

`src/modules/pinia.ts` 当前通过 `ids` 选择 store，通过 `paths` 选择字段：

```ts
app.use(
  installPiniaStorage({
    key: STORAGE_KEY, // '_pinia_storage'
    version: STORAGE_VERSION,
    ids: ['user', 'space', 'deploy-env'], // 持久化的 store id
    paths: ['statusTab', 'lastAppTemplateID', 'currentEnv'], // 持久化的字段
  }),
);
```

当前实现有两个必须注意的限制：

- `lastAppTemplateID` 属于 id 为 `app` 的 store，但 `ids` 未包含 `app`，所以该字段当前不会持久化
- 多个 store 共用同一个 storage key，任一 store 更新时都会用当前 store 的裁剪结果覆盖已保存对象，跨 store 字段不能可靠共存

在持久化插件改为“按 store 分组存储并合并已有状态”之前，不要仅通过追加 `ids` / `paths` 扩展持久化范围。

### 5.3 Store 使用位置

| 类型               | 位置                          |
| ------------------ | ----------------------------- |
| 全局状态           | `src/stores/`                 |
| 页面级状态         | 页面内 `ref` / `reactive`     |
| 跨组件共享但非全局 | `src/composables/` 组合式函数 |

## 6. 组合式函数 (Composables)

### 6.1 命名与放置

- **命名**：`use-xxx.ts`，导出 `useXxx` 函数
- **全局共享**：`src/composables/`
- **页面级**：`src/pages/{模块}/use-xxx.ts`（如 `deploy/use-deploy.ts`）

### 6.2 示例

```ts
// src/composables/use-my-feature.ts
import { ref, onMounted, onUnmounted } from 'vue';

export function useMyFeature() {
  const data = ref<string[]>([]);

  function update(newData: string[]) {
    data.value = newData;
  }

  return { data, update };
}
```

## 7. 样式

### 7.1 UnoCSS（原子化 CSS）

优先使用 UnoCSS 原子类：

```vue
<div class="flex items-center gap-2 p-4 rounded-lg bg-[#F5F7FA]">
  <span class="text-sm text-gray-600">文本</span>
</div>
```

### 7.2 全局样式

- `src/styles/main.css` — 主样式（`main.ts` 引入）
- `src/styles/bk-patch.css` — bkui-vue 样式补丁

### 7.3 组件样式

使用 `<style scoped>` 限定作用域：

```vue
<style scoped>
  .my-component {
    @apply flex items-center;
  }
</style>
```

## 8. 国际化 (i18n)

- 语言文件：`locales/zh-CN.yml`、`locales/en-US.yml`
- 模板中：`{{ $t('key') }}`
- Script 中：`window.i18n.t('key')`
- 新增文案同步添加到两个语言文件

## 9. API 定义

- API 名称与后端 proto 定义保持一致
- 自定义请求和返回数据结构
- 路径参数用 `{var}` 格式
- v1 API 通过 `pnpm gen:api:v1` 自动生成，勿手动编辑

## 10. 常量与枚举

| 类型     | 位置                                              |
| -------- | ------------------------------------------------- |
| 全局常量 | `src/common/const.ts`                             |
| 枚举     | `src/common/enums/`（如 `deploy.ts`、`build.ts`） |
| 正则校验 | `src/common/const.ts` 中的 `BKMS_REGEX`           |

> 不要在页面中散落定义常量，统一放到 `common/` 下管理。

## 11. TypeScript 配置

```json
// tsconfig.json 关键配置
{
  "compilerOptions": {
    "target": "ESNext",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "strictNullChecks": true,
    "noUnusedLocals": true,
    "paths": { "~/*": ["src/*"] }
  }
}
```

- **strict 模式**：严格类型检查
- **noUnusedLocals**：禁止未使用的局部变量
- 路径别名：`~/` → `src/`

## 12. Lint 工具链

| 工具      | 配置文件               | 用途                                                        |
| --------- | ---------------------- | ----------------------------------------------------------- |
| ESLint    | `eslint.config.mjs`    | JS/TS/Vue 代码检查；License header 自动检查仅覆盖 JS/TS/TSX |
| Biome     | `biome.json`           | 代码格式化                                                  |
| Prettier  | `prettier.config.mjs`  | 代码格式化                                                  |
| Stylelint | `stylelint.config.mjs` | CSS/SCSS/Vue 样式检查                                       |

```bash
pnpm lint          # ESLint 检查
pnpm lint:fix      # ESLint 修复
pnpm biome:format  # Biome 格式化
pnpm format        # Prettier 格式化
pnpm stylelint     # Stylelint 检查
```
