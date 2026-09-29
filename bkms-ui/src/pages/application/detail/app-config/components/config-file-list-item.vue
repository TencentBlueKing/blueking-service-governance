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
  <div
    class="group cursor-pointer border-l-[2px] px-[12px] py-[8px]"
    :class="active ? 'border-[#3A84FF] bg-[#E1ECFF]' : 'border-transparent hover:bg-[#F5F7FA]'"
    @click="emit('click')"
  >
    <div class="relative flex min-w-0 items-center pr-[20px]">
      <OverflowTitle
        class="min-w-0 text-[12px] text-[#313238]"
        type="tips"
      >
        {{ file.displayName || '--' }}
      </OverflowTitle>
      <div
        v-if="deletable"
        class="absolute right-0 top-1/2 flex -translate-y-1/2 items-center"
        :class="deleting ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'"
        @click.stop
      >
        <PopConfirm
          :title="$t('确认删除该配置文件？')"
          trigger="click"
          :width="300"
          @cancel="deleting = false"
          @confirm="handleDelete"
        >
          <template #content>
            <div class="mb-[4px]">
              <span>{{ $t('文件名') }}：</span>
              <span class="text-[#313238]">{{ file.displayName || '--' }}</span>
            </div>
            <div>{{ $t('删除后不再挂载到容器内，文件内容与历史版本一并删除，需要重新部署后生效。') }}</div>
          </template>
          <Del
            class="cursor-pointer p-[3px] text-[14px] text-[#979BA5] hover:text-[#EA3636]"
            @click="deleting = true"
          />
        </PopConfirm>
      </div>
    </div>
    <OverflowTitle
      class="mt-[2px] min-w-0 text-[12px] text-[#979BA5]"
      type="tips"
    >
      {{ file.displayPath || '--' }}
    </OverflowTitle>
    <div class="mt-[4px] flex flex-wrap items-center gap-[4px]">
      <Tag size="small">{{ scopeLabel }}</Tag>
      <Tag
        v-if="!file.isUnifiedConfig"
        size="small"
        theme="info"
        >{{ $t('按环境配置') }}</Tag
      >
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue';

  import { OverflowTitle, PopConfirm, Tag } from 'bkui-vue';
  import { Del } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';

  import type { ConfigFileListItem } from '~/pages/application/detail/app-config/use-config-file-defs';

  const props = withDefaults(
    defineProps<{
      active?: boolean;
      deletable?: boolean;
      file: ConfigFileListItem;
    }>(),
    {
      active: false,
      deletable: false,
    },
  );

  const emit = defineEmits<{
    click: [];
    delete: [];
  }>();

  const { t } = useI18n();
  const deleting = ref(false);
  const scopeLabel = computed(() => {
    if (props.file.configKind === 'framework' || props.file.mountScope === 'all') return t('全部环境');
    return t('指定 {0} 个', [props.file.mountedEnvNames.length]);
  });

  function handleDelete() {
    deleting.value = false;
    emit('delete');
  }
</script>
