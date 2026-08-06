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
  <BkmsContent
    :collapsible="true"
    :edit-disabled="!currentEnv?.isDefault"
    :edit-disabled-tips="$t('全局配置跨环境共享，不支持按环境修改')"
    :show-edit-icon="!isEditing"
    @edit="handleEdit"
  >
    <template #title>
      <div class="flex items-center">
        <div class="text-[14px]">{{ $t('启动配置') }}</div>
        <Tag
          class="ml-[6px] font-400 text-[10px] h-[20px] leading-[20px]"
          type="stroke"
        >
          {{ $t('全局') }}
        </Tag>
      </div>
    </template>
    <div class="bg-[#FFF] p-[16px]">
      <!-- 查看态 -->
      <template v-if="!isEditing">
        <div class="grid grid-cols-2 gap-[12px] gap-y-2">
          <FieldItem
            class="min-h-[20px]"
            :class="[appModelSpec?.command?.length ? '!items-start' : '']"
            container-height="auto"
            :field-value="$t('启动命令')"
            :field-width="205"
          >
            <template #value>
              <div class="flex flex-col gap-[4px] w-full min-w-0">
                <div
                  v-for="item in appModelSpec?.command || []"
                  :key="item"
                >
                  <OverflowTitle type="tips">{{ item }}</OverflowTitle>
                </div>
                <span
                  v-if="!appModelSpec?.command?.length"
                  class="text-[12px] text-[#313238]"
                  >--</span
                >
              </div>
            </template>
          </FieldItem>
          <FieldItem
            class="min-h-[20px]"
            :class="[appModelSpec?.args?.length ? '!items-start' : '']"
            container-height="auto"
            :field-value="$t('命令参数')"
            :field-width="205"
          >
            <template #value>
              <div class="flex flex-col gap-[4px] w-full min-w-0">
                <div
                  v-for="item in appModelSpec?.args || []"
                  :key="item"
                  class="w-full overflow-hidden"
                >
                  <OverflowTitle type="tips">{{ item }}</OverflowTitle>
                </div>
                <span
                  v-if="!appModelSpec?.args?.length"
                  class="text-[12px] text-[#313238]"
                  >--
                </span>
              </div>
            </template>
          </FieldItem>
        </div>
      </template>
      <!-- 编辑态 -->
      <div v-else>
        <div class="flex items-start gap-x-[24px]">
          <div class="!w-[400px]">
            <div class="mb-[6px] text-[14px] text-[#4D4F56]">{{ $t('启动命令') }}</div>
            <RepeatableInput
              ref="commandRef"
              v-model="editForm.command"
              required
              trim-on-input
            />
          </div>
          <div class="!w-[400px]">
            <div class="mb-[6px] text-[14px] text-[#4D4F56]">{{ $t('命令参数') }}</div>
            <RepeatableInput
              ref="argsRef"
              v-model="editForm.args"
              required
              trim-on-input
            />
          </div>
        </div>

        <div class="!mb-0 mt-[16px] flex items-center">
          <Button
            :loading="saving"
            theme="primary"
            @click="handleSave"
          >
            {{ $t('保存') }}
          </Button>
          <Button
            class="ml-[8px]"
            @click="handleCancel"
          >
            {{ $t('取消') }}
          </Button>
        </div>
      </div>
    </div>
  </BkmsContent>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue';

  import { Button, Message, OverflowTitle, Tag } from 'bkui-vue';
  import { cloneDeep, set } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { AppModelSpecInput } from '~/@types/v1/app';
  import BkmsContent from '~/components/bkms-content.vue';
  import FieldItem from '~/components/field-item.vue';
  import RepeatableInput from '~/components/repeatable-input.vue';
  import useSpecField from '~/composables/use-spec-field';
  import { useAppDetail } from '~/stores/app-detail';

  import type { ExtendedEnv } from './types';

  interface Props {
    currentEnv: ExtendedEnv | null;
  }

  defineProps<Props>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const { updateSpecApi } = useSpecField();

  // 从 store 获取 appModelSpec
  const appModelSpec = computed(() => appDetailStore.appDetail?.appModelSpec);

  // ========== 编辑态控制 ==========
  const isEditing = ref(false);
  const saving = ref(false);
  const commandRef = ref<InstanceType<typeof RepeatableInput> | null>(null);
  const argsRef = ref<InstanceType<typeof RepeatableInput> | null>(null);

  // 编辑表单（编辑态使用的本地副本）
  const editForm = ref<{ args: string[]; command: string[] }>({
    command: [],
    args: [],
  });

  // 编辑前快照（用于取消时恢复）
  let snapshot: null | { args: string[]; command: string[] } = null;

  /** 取消编辑 */
  function handleCancel() {
    if (snapshot) {
      editForm.value = { ...snapshot };
    }
    isEditing.value = false;
  }

  /** 进入编辑态 */
  function handleEdit() {
    const currentCommand = appModelSpec.value?.command || [];
    const currentArgs = appModelSpec.value?.args || [];
    editForm.value = {
      command: [...currentCommand],
      args: [...currentArgs],
    };
    snapshot = {
      command: [...currentCommand],
      args: [...currentArgs],
    };
    isEditing.value = true;
  }

  /** 保存编辑 */
  async function handleSave() {
    const appDetail = appDetailStore.appDetail;
    if (!appDetail) return;

    // 校验必填项：当字段为空时无需校验
    const commandValid = editForm.value.command.length === 0 ? true : await commandRef.value?.validate();
    const argsValid = editForm.value.args.length === 0 ? true : await argsRef.value?.validate();

    if (!commandValid || !argsValid) return;

    saving.value = true;
    const updatedAppModelSpec = cloneDeep(appDetail.appModelSpec || {}) as AppModelSpecInput;
    set(updatedAppModelSpec, 'command', editForm.value.command);
    set(updatedAppModelSpec, 'args', editForm.value.args);

    const result = await updateSpecApi
      .value({
        appID: appDetailStore.appID,
        appModelSpec: updatedAppModelSpec,
      })
      .then(() => true)
      .catch(() => false);

    if (result) {
      Message({
        message: t('操作成功'),
        theme: 'success',
      });
      await appDetailStore.fetchAppDetail();
      isEditing.value = false;
    }
    saving.value = false;
  }
</script>

<style lang="postcss" scoped>
  :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
</style>
