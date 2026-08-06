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
  <template
    v-for="group in cardGroups"
    :key="group.key"
  >
    <!-- 分组标题（仅 AllData 模式下显示） -->
    <ToggleCard
      v-if="group.title && group.items.length"
      class="mt-[16px]"
      content-class="!pt-0 mt-[16px] overflow-visible"
      :name="`${group.title}（${group.items.length}）`"
      normal-bg-color="#EAEBF0"
      type="normal"
    >
      <!-- 卡片网格 -->
      <div
        v-if="group.items.length"
        class="card-grid"
      >
        <CardItem
          v-for="item in group.items"
          :key="item.name"
          :footer-type="getCardFooterType(item)"
          :item="item"
          @delete="handleDelete"
          @edit="handleEdit"
        />
      </div>
    </ToggleCard>
    <div
      v-else
      class="card-grid"
    >
      <CardItem
        v-for="item in group.items"
        :key="item.name"
        :footer-type="getCardFooterType(item)"
        :item="item"
        @delete="handleDelete"
        @edit="handleEdit"
      />
    </div>
  </template>
  <slot name="empty"></slot>
</template>
<script setup lang="ts">
  import { computed, h } from 'vue';

  import { InfoBox } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { type ComponentDefOutputObj } from '~/@types/v1/component-defs';

  import CardItem from './card-item.vue';
  import { type CardFooterType } from './card-item.vue';
  import { type ProcessedDataKey, ProcessedDataType } from './space-component.vue';

  interface IProps {
    data: ProcessedDataType;
    type: ProcessedDataKey;
  }
  const props = defineProps<IProps>();
  const emits = defineEmits(['edit', 'delete']);

  const { t } = useI18n();

  interface CardGroup {
    items: ComponentDefOutputObj[];
    key: string;
    title: string;
  }

  /** 根据 type 生成分组列表，统一 AllData 和非 AllData 的渲染逻辑 */
  const cardGroups = computed<CardGroup[]>(() => {
    if (props.type === 'AllData') {
      return [
        {
          key: 'personal',
          title: props.data.PersonalSpaceData.label,
          items: props.data.PersonalSpaceData.value,
        },
        {
          key: 'builtin-shared',
          title: t('内置 & 共享'),
          items: [...props.data.ShareData.value, ...props.data.BuiltinData.value],
        },
      ];
    }
    return [
      {
        key: props.type,
        title: '',
        items: props.data[props.type].value,
      },
    ];
  });

  /** 根据卡片数据判断底部区域类型 */
  function getCardFooterType(item: ComponentDefOutputObj): CardFooterType {
    if (item.isBuiltin) return 'builtin';
    if (props.data.PersonalSpaceData.value.includes(item)) return 'editable';
    return 'shared';
  }

  function handleDelete(item: ComponentDefOutputObj) {
    if ((item.appCompInstanceCount ?? 0) > 0) return;
    InfoBox({
      title: `${t('确认删除该组件')}?`,
      headerAlign: 'center',
      footerAlign: 'center',
      content: h('div', [
        h('div', [t('组件名称: {name}', { name: item.name || item.displayName })]),
        h('div', { class: 'mt-[10px]' }, [t('删除后，将不可恢复，请谨慎操作！')]),
      ]),
      confirmButtonTheme: 'danger',
      confirmText: t('删除'),
      cancelText: t('取消'),
      async onConfirm() {
        emits('delete', item);
      },
    });
  }
  function handleEdit(item: ComponentDefOutputObj) {
    emits('edit', item);
  }
</script>
<style scoped>
  .card-grid {
    display: grid;
    gap: 16px;
    grid-auto-rows: max-content;
    grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  }
</style>
