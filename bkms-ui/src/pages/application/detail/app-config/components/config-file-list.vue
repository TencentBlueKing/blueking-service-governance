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
  <div class="flex h-full w-full flex-col bg-[#fff]">
    <div class="flex shrink-0 items-center border-b border-[#EAEBF0] px-[12px] py-[16px]">
      <span class="text-[14px] font-bold leading-[22px] text-[#313238]">{{ $t('文件列表') }}</span>
      <a
        v-bk-tooltips="$t('点击查看配置文件说明')"
        :aria-label="$t('点击查看配置文件说明')"
        class="ml-[6px] flex items-center text-[16px] text-[#3A84FF]"
        :href="configFileDocUrl"
        rel="noopener noreferrer"
        target="_blank"
      >
        <InfoLine />
      </a>
    </div>
    <div class="flex flex-1 flex-col overflow-auto py-[8px]">
      <Skeleton
        :full-height="false"
        :loading="loading"
      >
        <template #loading>
          <div class="px-[12px] py-[6px]">
            <Layout.shape
              :height="16"
              :width="88"
            />
          </div>
          <div class="px-[12px] py-[8px]">
            <Layout.shape
              :height="18"
              :width="120"
            />
            <div class="mt-[6px]">
              <Layout.shape
                :height="16"
                :width="160"
              />
            </div>
            <div class="mt-[6px]">
              <Layout.shape
                :height="18"
                :width="68"
              />
            </div>
          </div>
          <div class="mt-[8px] flex items-center justify-between px-[12px] py-[6px]">
            <Layout.shape
              :height="16"
              :width="108"
            />
            <Layout.shape
              :height="16"
              :width="48"
            />
          </div>
          <div
            v-for="index in 5"
            :key="index"
            class="px-[12px] py-[8px]"
          >
            <Layout.shape
              :height="18"
              :width="120"
            />
            <div class="mt-[6px]">
              <Layout.shape
                :height="16"
                :width="160"
              />
            </div>
            <div class="mt-[6px]">
              <Layout.shape
                :height="18"
                :width="68"
              />
            </div>
          </div>
        </template>
        <div class="px-[12px] py-[6px] text-[12px] text-[#979BA5]">{{ $t('框架配置文件') }}</div>
        <FileItem
          v-for="file in frameworkFiles"
          :key="file.id"
          :active="file.id === activeId"
          :file="file"
          @click="emit('select', file.id)"
        />

        <div class="mt-[8px] flex items-center justify-between px-[12px] py-[6px]">
          <span class="text-[12px] text-[#979BA5]">{{ $t('其他配置文件') }}（{{ plainFiles.length }}）</span>
          <Button
            :disabled="createDisabled"
            text
            theme="primary"
            @click="emit('create')"
          >
            <Plus
              :height="20"
              :width="20"
            />
            {{ $t('新建') }}
          </Button>
        </div>
        <FileItem
          v-for="file in plainFiles"
          :key="file.id"
          :active="file.id === activeId"
          deletable
          :file="file"
          @click="emit('select', file.id)"
          @delete="emit('delete', file)"
        />
        <div
          v-if="!plainFiles.length"
          class="flex flex-1 items-center justify-center px-[12px] py-[24px]"
        >
          <Exception
            class="!h-auto"
            scene="part"
            type="empty"
          >
            <template #title><span /></template>
            <template #description>
              <span class="text-[12px] leading-[20px] text-[#979BA5]">{{ $t('暂无其他配置文件，可点击新建') }}</span>
            </template>
          </Exception>
        </div>
      </Skeleton>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Button, Exception } from 'bkui-vue';
  import { InfoLine, Plus } from 'bkui-vue/lib/icon';
  import { DOC_LINKS } from '~/common/const';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import FileItem from '~/pages/application/detail/app-config/components/config-file-list-item.vue';

  import type { ConfigFileListItem } from '~/pages/application/detail/app-config/use-config-file-defs';

  const props = withDefaults(
    defineProps<{
      activeId?: string;
      createDisabled?: boolean;
      files: ConfigFileListItem[];
      loading?: boolean;
    }>(),
    { activeId: '', createDisabled: false, loading: false },
  );

  const emit = defineEmits<{
    create: [];
    delete: [file: ConfigFileListItem];
    select: [id: string];
  }>();

  const frameworkFiles = computed(() => props.files.filter(file => file.configKind === 'framework'));
  const plainFiles = computed(() => props.files.filter(file => file.configKind === 'plain'));
  const configFileDocUrl = `${import.meta.env.BK_DOC_URL}${DOC_LINKS.CONFIG_FILE_GUIDE}`;
</script>
