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
  <Dialog
    v-model:is-show="isShow"
    :width="640"
    @hidden="handleHidden"
  >
    <template #header>
      <div class="flex flex-col items-center">
        <SvgIcon
          :height="42"
          icon="bkms-icon-tishi"
          :width="42"
        />
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ $t('清空文件内容') }}
        </span>
      </div>
    </template>

    <div class="text-[14px] text-[#313238] mb-[16px] mt-[32px]">
      {{ $t('文件清空后，该环境将恢复为默认配置。请选择处理方式：') }}
    </div>

    <Radio.Group
      v-model="selectedAction"
      class="clear-file-action-group flex flex-col gap-[12px]"
    >
      <div
        v-for="item in actionOptions"
        :key="item.value"
        class="clear-file-action-card cursor-pointer rounded-[4px] border border-solid border-[#dcdee5] px-[16px] py-[14px] transition-all duration-300"
        :class="{ 'is-active': selectedAction === item.value }"
        @click="selectedAction = item.value"
      >
        <Radio :label="item.value">
          <div class="ml-[8px]">
            <div class="font-700 text-[12px] text-[#313238]">{{ item.title }}</div>
            <div class="mt-[8px] text-[12px] leading-[20px] text-[#63656E]">
              {{ item.desc }}
              <template v-if="item.value === 'saveEmpty'">
                <br />
                {{ $t('环境列表中仍会标记为：') }}
                <span class="modified-status text-[#FF9C01]">{{ $t('已修改') }}</span>
              </template>
            </div>
          </div>
        </Radio>
      </div>
    </Radio.Group>

    <template #footer>
      <div class="flex justify-center">
        <Button
          class="mr-[8px]"
          :loading="loading"
          theme="primary"
          @click="handleConfirm"
        >
          {{ $t('确认保存') }}
        </Button>
        <Button @click="handleHidden">
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Button, Dialog, Radio } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import SvgIcon from '~/components/svg-icon.vue';

  type ClearFileContentAction = 'deleteFile' | 'saveEmpty';

  interface Props {
    loading?: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    loading: false,
  });

  const emit = defineEmits<{ (e: 'confirm', action: ClearFileContentAction): void }>();

  const { t } = useI18n();
  const isShow = defineModel<boolean>('isShow', { default: false });
  const selectedAction = ref<ClearFileContentAction>('saveEmpty');

  const actionOptions = computed(() => [
    {
      value: 'saveEmpty' as const,
      title: t('保存为空文件'),
      desc: t('保留环境下差异配置文件，可查看历史版本、随时回滚。'),
    },
    {
      value: 'deleteFile' as const,
      title: t('删除文件'),
      desc: t('删除环境下差异配置文件，历史版本一起删除，不可恢复、回滚。'),
    },
  ]);

  function handleConfirm() {
    if (props.loading) return;
    emit('confirm', selectedAction.value);
  }

  function handleHidden() {
    isShow.value = false;
    selectedAction.value = 'saveEmpty';
  }
</script>

<style lang="postcss" scoped>
  .clear-file-action-group {
    :deep(.bk-radio-group) {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
  }

  .clear-file-action-card {
    &:hover {
      border-color: #3a84ff;
    }

    &.is-active {
      border-color: #3a84ff;
      background: #f5f7fa;
      color: #3a84ff !important;

      :deep(.bk-radio-label),
      :deep(.bk-radio-label *) {
        color: #3a84ff !important;
      }

      .modified-status {
        color: #ff9c01 !important;
      }
    }

    :deep(.bk-radio) {
      align-items: flex-start;
    }

    :deep(.bk-radio-input) {
      margin-top: 2px;
    }

    :deep(.bk-radio-label) {
      font-size: 12px;
    }
  }

  :deep(.bk-dialog-header) {
    padding-top: 48px;
  }

  :deep(.bk-dialog-content) {
    padding: 0 32px;
  }

  :deep(.bk-dialog-footer) {
    border: none;
    background-color: unset;
    padding-bottom: 24px;
    padding-top: 0;
  }
</style>
