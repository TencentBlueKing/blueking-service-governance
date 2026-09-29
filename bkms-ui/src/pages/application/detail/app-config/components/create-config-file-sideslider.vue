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
  <Sideslider
    v-model:is-show="visible"
    :width="720"
  >
    <template #header>
      <DividerHeader
        :show-divider="!!formData.name"
        :title="$t('新建配置文件')"
        :title-size="16"
      >
        <span
          v-if="formData.name"
          class="truncate"
          >{{ formData.name }}</span
        >
      </DividerHeader>
    </template>
    <div
      class="overflow-y-auto px-[24px] pb-[32px] pt-[20px]"
      style="max-height: calc(100vh - 106px)"
    >
      <Form
        ref="formRef"
        form-type="vertical"
        :model="formData"
        :rules="rules"
      >
        <Form.FormItem
          :label="$t('文件名')"
          property="name"
          required
        >
          <Input
            v-model.trim="formData.name"
            :maxlength="64"
            :placeholder="$t('请输入由字母、数字、短横线或下划线组成的文件名')"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('配置文件路径')"
          property="mountDir"
          required
        >
          <Input
            v-model.trim="formData.mountDir"
            clearable
            :placeholder="$t('请输入容器内目录，例如 /data/conf')"
          />
          <div class="mt-[4px] text-[12px] text-[#979BA5]">
            {{ $t('文件最终挂载到 {0}', [`${formData.mountDir || '/[路径]'}/${formData.name || '[文件名]'}`]) }}
          </div>
        </Form.FormItem>
        <Form.FormItem
          :label="$t('挂载环境')"
          property="mountedEnvNames"
          :required="formData.scope === 'envs'"
        >
          <MountScopeField
            v-model:env-names="formData.mountedEnvNames"
            v-model:scope="formData.scope"
            :env-list="envList"
            @update:scope="handleScopeChange"
          />
        </Form.FormItem>
        <Form.FormItem :label="$t('渲染环境变量')">
          <div class="flex items-center">
            <Switcher
              v-model="formData.enableEnvVarRender"
              size="small"
              theme="primary"
            />
            <span class="ml-[10px] text-[12px] text-[#979BA5]">{{ $t('开启后，环境变量占位符会在下发前被替换') }}</span>
          </div>
        </Form.FormItem>
      </Form>
    </div>
    <template #footer>
      <Button
        :loading="loading"
        theme="primary"
        @click="handleSubmit"
        >{{ $t('确定') }}</Button
      >
      <Button
        class="ml-[8px]"
        @click="visible = false"
        >{{ $t('取消') }}</Button
      >
    </template>
  </Sideslider>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue';

  import { Button, Form, Input, Sideslider, Switcher } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { BKMS_REGEX } from '~/common/const';
  import MountScopeField from '~/pages/application/detail/app-config/components/mount-scope-field.vue';

  import type { EnvOutput } from '~/@types/v1/env';
  import type { MountScope } from '~/pages/application/detail/app-config/components/mount-scope-field.vue';
  import type { PlainFileCreateInput } from '~/pages/application/detail/app-config/use-config-file-defs';

  const props = withDefaults(
    defineProps<{
      /** 可选挂载环境列表 */
      envList: EnvOutput[];
      isShow: boolean;
      /** 提交中状态（父组件控制，用于按钮 loading） */
      loading?: boolean;
    }>(),
    { loading: false },
  );
  const emit = defineEmits<{
    /** 提交创建参数，由父组件完成实际创建 */
    submit: [value: PlainFileCreateInput];
    'update:isShow': [value: boolean];
  }>();
  const { t } = useI18n();

  const visible = computed({
    get: () => props.isShow,
    set: value => emit('update:isShow', value),
  });
  /** 表单数据 */
  const formData = reactive({
    /** 是否开启环境变量渲染 */
    enableEnvVarRender: false,
    /** 容器内挂载目录 */
    mountDir: '',
    /** 指定环境下的环境名集合 */
    mountedEnvNames: [] as string[],
    /** 文件名 */
    name: '',
    /** 挂载范围 */
    scope: 'all' as MountScope,
  });

  /** 表单 ref */
  const formRef = ref<InstanceType<typeof Form> | null>(null);

  /** 表单校验规则 */
  const rules = {
    name: [
      {
        message: t('文件名仅支持字母、数字、短横线和下划线，长度 1-64'),
        trigger: 'blur',
        validator: (val: string) => BKMS_REGEX.appConfigFileNameRegex.test(val),
      },
    ],
    mountDir: [
      {
        message: t('配置文件路径必须以 / 开头，且不能为 / 或以 / 结尾'),
        trigger: 'blur',
        validator: (val: string) => BKMS_REGEX.appConfigMountDirRegex.test(val),
      },
    ],
    mountedEnvNames: [
      {
        message: t('请至少选择一个挂载环境'),
        trigger: 'change',
        validator: () => formData.scope !== 'envs' || formData.mountedEnvNames.length > 0,
      },
    ],
  };

  function handleScopeChange(scope: MountScope) {
    if (scope === 'all') formRef.value?.clearValidate('mountedEnvNames');
  }

  /** 校验表单并触发 submit 事件 */
  async function handleSubmit() {
    const valid = await formRef.value
      ?.validate()
      .then(() => true)
      .catch(() => false);
    if (!valid) return;

    emit('submit', {
      enableEnvVarRender: formData.enableEnvVarRender,
      mountDir: formData.mountDir,
      // 全部环境不传该字段；指定环境时传所选环境名集合。
      mountedEnvNames: formData.scope === 'all' ? undefined : [...formData.mountedEnvNames],
      name: formData.name,
    });
  }

  // 每次打开抽屉时重置表单为默认值
  watch(
    () => props.isShow,
    value => {
      if (!value) return;
      Object.assign(formData, {
        enableEnvVarRender: false,
        mountDir: '',
        mountedEnvNames: [],
        name: '',
        scope: 'all',
      });
      formRef.value?.clearValidate();
    },
  );
</script>

<style lang="postcss" scoped>
  :deep(.bk-sideslider-footer) {
    margin-top: 0;
  }
</style>
