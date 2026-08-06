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
  <div class="min-h-[200px]">
    <Exception
      scene="part"
      :title="typeTitleMap[type]"
      :type="typeMap[type]"
    >
      <template
        v-if="$slots.type"
        #type
      >
        <slot name="type"></slot>
      </template>
      <i18n-t
        v-if="type === 'search'"
        keypath="可以尝试 调整关键词 或 {0}"
      >
        <Button
          text
          theme="primary"
          @click="handleClearFilter"
          >{{ $t('清空筛选条件') }}</Button
        >
      </i18n-t>
      <Button
        v-else-if="type === 'error'"
        text
        theme="primary"
        @click="handleRefresh"
      >
        {{ $t('刷新') }}
      </Button>
    </Exception>
  </div>
</template>

<script lang="ts" setup>
  import { Button, Exception } from 'bkui-vue';
  import { ExceptionEnum } from 'bkui-vue/lib/exception';
  import { useI18n } from 'vue-i18n';

  interface IProps {
    type?: TableEmptyType;
  }
  type TableEmptyType = 'empty' | 'error' | 'search';

  withDefaults(defineProps<IProps>(), {
    type: 'empty',
  });

  const emit = defineEmits(['clear', 'refresh']);

  const { t } = useI18n();
  const typeTitleMap: Record<TableEmptyType, string> = {
    empty: t('暂无数据'),
    search: t('搜索结果为空'),
    error: t('数据获取异常'),
  };
  const typeMap: Record<TableEmptyType, ExceptionEnum> = {
    empty: ExceptionEnum.EMPTY,
    search: ExceptionEnum.SEARCH,
    error: ExceptionEnum.CODE_500,
  };

  function handleClearFilter() {
    emit('clear');
  }

  function handleRefresh() {
    emit('refresh');
  }
</script>

<style lang="postcss" scoped>
  :deep(.bk-exception-img) {
    width: 220px;
    height: 120px;
  }
  :deep(.bk-exception-title) {
    font-size: 14px;
    margin-top: 0px;
  }
</style>
