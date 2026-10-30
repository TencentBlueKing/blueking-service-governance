<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<template>
  <Navigation
    :class="['h-full', theme, { 'hide-title': !needTitle, 'no-header-page': !showDefaultHeader }]"
    :default-open="navigationDefaultOpen"
    hover-width="240"
    v-bind="$attrs"
    @toggle-click="navigationState.setOpen"
  >
    <template #menu>
      <Menu
        v-model:active-key="activeKey"
        v-bkloading="{ loading }"
        class="min-h-[50vh]"
        :opened-keys="openedKeys"
      >
        <template v-for="item in list">
          <!-- 导航组 -->
          <Menu.Group
            v-if="'foldName' in item"
            :key="`group-${item.key}`"
            :fold-name="item.foldName"
            :name="item.name"
          >
            <Menu.Item
              v-for="child in item.children"
              :key="child.key"
              :disabled="child.disabled"
            >
              {{ child.name }}
              <template
                v-if="child.icon"
                #icon
              >
                <i :class="`bkms-icon bkms-icon-${child.icon}`"></i>
              </template>
            </Menu.Item>
          </Menu.Group>
          <!-- 可展开的菜单 -->
          <Menu.Submenu
            v-else-if="'title' in item"
            :key="`sub-${item.key}`"
            :title="item.title"
          >
            <Menu.Item
              v-for="child in item.children"
              :key="child.key"
              :disabled="child.disabled"
            >
              {{ child.name }}
              <template
                v-if="child.icon"
                #icon
              >
                <i :class="`bkms-icon bkms-icon-${child.icon}`"></i>
              </template>
            </Menu.Item>
          </Menu.Submenu>
          <Menu.Item
            v-else
            :key="item.key"
            :disabled="item.disabled"
          >
            {{ item.name }}
            <template
              v-if="item.icon"
              #icon
            >
              <i :class="`bkms-icon bkms-icon-${item.icon}`"></i>
            </template>
          </Menu.Item>
        </template>
      </Menu>
    </template>
    <slot></slot>
    <template #side-header>
      <slot name="side-header"></slot>
    </template>
    <template #side-icon>
      <slot name="side-icon"></slot>
    </template>
    <template
      v-if="showDefaultHeader"
      #header
    >
      <slot name="header">
        <span class="text-[16px]">
          {{
            activeItem && 'name' in activeItem
              ? activeItem.name
              : activeItem && 'title' in activeItem
                ? activeItem.title
                : ''
          }}
        </span>
      </slot>
    </template>
  </Navigation>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Menu, Navigation } from 'bkui-vue';
  import { cloneDeep } from 'lodash-es';
  import usePersistentStorage from '~/composables/use-persistent-storage';

  import type { NavigationItem } from '~/config/navigation/types';

  interface DefaultOpenConfig {
    /** 是否强制使用value，若为true，则使用传入的value，否则使用存储的值 */
    force?: boolean;
    /** 是否默认展开 */
    value?: boolean;
  }

  const activeKey = defineModel<string>('activeKey', { required: true });

  const props = withDefaults(
    defineProps<{
      bg?: string;
      defaultOpen?: DefaultOpenConfig;
      list: Array<NavigationItem>;
      needTitle?: boolean;
      openedKeys?: string[];
      theme?: 'dark' | 'light';
    }>(),
    {
      bg: '#F0F4FA',
      defaultOpen: () => ({ value: true }),
      needTitle: true,
      theme: 'light',
      openedKeys: () => [],
    },
  );

  // 使用导航存储hooks，支持持久化展开状态
  const { useNavigation } = usePersistentStorage();
  const navigationState = useNavigation(props.defaultOpen, true);

  const navigationDefaultOpen = computed(() => {
    return props.defaultOpen?.force ? props.defaultOpen?.value : navigationState.isOpen.value;
  });

  const loading = ref(true);

  const formatList = computed(() =>
    cloneDeep(props.list).reduce((acc, cur) => {
      if ('children' in cur) {
        acc.push(...cur.children);
      } else {
        acc.push(cur);
      }
      return acc;
    }, [] as NavigationItem[]),
  );

  watch(
    () => props.list,
    newList => {
      // 当菜单列表有数据时，关闭 loading
      if (newList && newList.length > 0) {
        loading.value = false;
      }
    },
    { deep: true, immediate: true },
  );

  const activeItem = computed(() => formatList.value.find(item => item.key === activeKey.value));

  // 当 layout 为 'empty' 时，不显示默认 header 的顶部导航
  const showDefaultHeader = computed(() => {
    // 先直接从 list 中查找当前 activeKey 对应的菜单项
    const currentItem = props.list
      .flatMap(item => ('children' in item ? item.children : [item]))
      .find(item => item.key === activeKey.value);

    // 如果找不到对应菜单项，默认显示 header
    if (!currentItem) {
      return true;
    }

    return currentItem.meta?.layout !== 'empty';
  });
</script>

<style lang="postcss" scoped>
  .light :deep(.bk-navigation-wrapper .navigation-nav .nav-slider),
  .light :deep(.bk-menu) {
    background-color: var(--aidev-vertical-nav-background);
  }

  .light :deep(.bk-navigation-wrapper .navigation-nav .nav-slider) {
    border-right: 1px solid var(--aidev-vertical-nav-border-color) !important;
  }

  /* 组件库菜单宽写死 260px，侧栏展开是 240px，不改成 100% 时左右 8px 外边距会被裁掉 */
  .light :deep(.bk-menu) {
    width: 100%;
  }

  .light :deep(.bk-menu-group .group-name) {
    height: 40px;
    margin: 0 24px;
    padding: 0;
    font-size: 12px;
    line-height: 40px;
    color: var(--aidev-vertical-nav-group-color) !important;
  }

  :deep(.bk-navigation-wrapper .navigation-container .container-header) {
    justify-content: start;
    color: #111214;
    box-shadow: 0 3px 4px 0 #0000000a;
    border-bottom: none;
    z-index: 1;
  }

  :deep(.bk-navigation-wrapper .navigation-container .container-content) {
    background-color: v-bind(bg);
    background-attachment: fixed;
  }

  .light :deep(.bk-menu .bk-menu-item) {
    box-sizing: border-box;
    width: calc(100% - 16px);
    height: 40px;
    min-height: 40px;
    margin: 0 8px;
    padding: 0 16px;
    border-radius: 8px;
    color: var(--aidev-vertical-nav-item-color);

    .item-icon {
      flex: 0 0 16px;
      margin-right: 16px;
      font-size: 16px;
    }

    i {
      color: var(--aidev-vertical-nav-icon-color);
    }

    .item-content {
      color: var(--aidev-vertical-nav-item-color);
    }

    &:not(.is-active):not(.is-disabled):hover {
      color: var(--aidev-vertical-nav-item-hover-color);
      background: var(--aidev-vertical-nav-item-hover-background) !important;
    }
  }

  /* 收起后侧栏 60px，菜单项仍留左右 8px。图标 16px 居中，不再铺满整列。 */
  .light :deep(.nav-slider[style*='width: 60px'] .bk-menu .bk-menu-item) {
    justify-content: center;
    width: calc(100% - 16px);
    margin: 2px 8px;
    padding: 0;
    border-radius: 8px;

    .item-icon {
      flex: 0 0 16px;
      margin-right: 0;
    }

    .item-content {
      display: none;
    }
  }

  .light :deep(.bk-menu .is-active) {
    background: var(--aidev-vertical-nav-item-active-background) !important;
    color: var(--aidev-vertical-nav-item-active-color) !important;

    .item-content {
      color: var(--aidev-vertical-nav-item-active-color);
    }

    /* 图标是字体，渐变要裁在 ::before 的字形上，色差拉大才看得出 */
    i {
      color: transparent;

      &::before {
        background-image: var(--aidev-vertical-nav-item-active-icon);
        -webkit-background-clip: text;
        background-clip: text;
        -webkit-text-fill-color: transparent;
        color: transparent;
      }
    }
  }

  .light :deep(.bk-navigation-title) {
    border-bottom: 1px solid var(--aidev-vertical-nav-title-border-color) !important;
  }

  :deep(.bk-navigation-title) {
    padding: 10px 8px;
  }

  .hide-title :deep(.bk-navigation-title) {
    display: none;
  }

  :deep(.bk-navigation-wrapper .navigation-nav .nav-slider-list) {
    padding: 8px 0;
    position: relative;
  }

  .light :deep(.bk-navigation-wrapper .navigation-nav .footer-icon) {
    color: var(--aidev-vertical-nav-icon-color);

    &:hover {
      background: var(--aidev-vertical-nav-item-hover-background) !important;
      color: var(--aidev-vertical-nav-collapse-hover-color) !important;
    }
  }

  /* 隐藏默认 header 区域 */
  .no-header-page :deep(.bk-navigation-wrapper .navigation-container .container-header) {
    display: none !important;
    height: 0 !important;
  }
</style>
