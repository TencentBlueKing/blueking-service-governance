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
    :width="480"
    @hidden="handleHidden"
  >
    <template #header>
      <div class="flex flex-col items-center">
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ $t('保存并创建新版本') }}
        </span>
      </div>
    </template>

    <div class="text-[14px] text-[#313238] mb-[16px] mt-[36px]">
      {{ $t('保存将生成新的版本号，并记录到版本历史中，可用于后续对比和回滚。') }}
    </div>

    <!-- 版本信息 -->
    <div class="bg-[#F5F7FA] mb-[16px] py-[12px] px-[16px] text-[12px]">
      <ul class="leading-[20px]">
        <li>
          <span class="text-[#4D4F56]">{{ $t('版本号') }}：</span>
          <span class="text-[#313238]">V{{ nextVersion }}</span>
        </li>
      </ul>
    </div>

    <!-- 版本描述 -->
    <Form
      ref="formRef"
      form-type="vertical"
      :model="formData"
      :rules="rules"
    >
      <Form.FormItem
        :label="$t('版本描述')"
        property="description"
        required
      >
        <Input
          v-model="formData.description"
          :placeholder="$t('请输入版本描述')"
          :rows="3"
          type="textarea"
        />
      </Form.FormItem>
    </Form>

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
  import { reactive, ref } from 'vue';

  import { Button, Dialog, Form, Input } from 'bkui-vue';

  interface Props {
    /** 下一版本号 = currentVersion + 1 */
    nextVersion?: number;
  }

  withDefaults(defineProps<Props>(), {
    nextVersion: 1,
  });

  const emit = defineEmits<{ (e: 'confirm', description: string): void }>();

  const isShow = defineModel<boolean>('isShow', { default: false });

  /** 表单数据 */
  const formData = reactive({
    description: '',
  });

  /** 表单校验规则 */
  const rules = {
    description: [
      {
        message: '请输入版本描述',
        required: true,
        trigger: 'blur',
        validator: (val: string) => val.trim().length > 0,
      },
    ],
  };

  /** 表单 ref */
  const formRef = ref<InstanceType<typeof Form> | null>(null);
  const loading = ref(false);

  /** 确认保存 */
  async function handleConfirm() {
    const valid = await formRef.value
      ?.validate()
      .then(() => true)
      .catch(() => false);
    if (!valid) return;

    loading.value = true;
    emit('confirm', formData.description.trim());
    // 由父组件负责关闭弹窗（在 handleSave 成功后）
  }

  /** 停止 loading */
  function stopLoading() {
    loading.value = false;
  }

  defineExpose({ stopLoading });

  /** 关闭弹窗 */
  function handleHidden() {
    isShow.value = false;
    loading.value = false;
    formData.description = '';
    formRef.value?.clearValidate();
  }
</script>

<style lang="postcss" scoped>
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
