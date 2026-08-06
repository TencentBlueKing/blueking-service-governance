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
    footer-align="right"
    :title="t('新建版本')"
    :width="560"
  >
    <Form
      ref="formRef"
      form-type="vertical"
      :model="formData"
    >
      <Form.FormItem
        :label="t('分支')"
        property="branch"
        required
      >
        <Input
          v-model.trim="formData.branch"
          :placeholder="t('请输入分支名')"
        />
      </Form.FormItem>
      <Form.FormItem
        :label="t('版本类型')"
        property="bumpType"
        required
      >
        <Radio.Group v-model="formData.bumpType">
          <Radio label="major">
            {{ t('重大版本') }}
          </Radio>
          <Radio label="minor">
            {{ t('次版本') }}
          </Radio>
          <Radio label="patch">
            {{ t('修正版本') }}
          </Radio>
        </Radio.Group>
      </Form.FormItem>
      <div class="relative">
        <Form.FormItem
          :label="t('版本号')"
          required
        >
          <Input
            :model-value="nextVersion ? `${nextVersion}（${t('主版本号.次版本号.修正版本号')}）` : ''"
            readonly
          />
        </Form.FormItem>
        <div class="absolute top-0 left-[60px] flex items-center">
          <i class="bkms-icon bkms-icon-warning-circle text-[14px] text-[#979BA5]"></i>
          <span class="text-[#979BA5] text-[12px] ml-[4px]">
            {{ $t('根据选择的版本类型自动生成') }}
          </span>
        </div>
      </div>
    </Form>
    <template #footer>
      <Button
        :loading="isSubmitting"
        theme="primary"
        @click="handleSubmit"
      >
        {{ t('确定') }}
      </Button>
      <Button
        class="ml-[8px]"
        @click="handleCancel"
      >
        {{ t('取消') }}
      </Button>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { nextTick, ref, watch } from 'vue';

  import { Button, Dialog, Form, Input, Message, Radio } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { HelmChartsService } from '~/api/modules/v1';
  import { useAppDetail } from '~/stores/app-detail';

  /** semver 递增段类型 */
  type BumpType = 'major' | 'minor' | 'patch';

  interface Emits {
    (e: 'update:modelValue', value: boolean): void;
    (e: 'success', version: string, buildID: string): void;
  }

  interface Props {
    appId: string;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();
  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  const isShow = defineModel<boolean>({ required: true });

  const formData = ref<{
    branch: string;
    bumpType: BumpType;
  }>({
    branch: '',
    bumpType: 'patch',
  });

  const nextVersion = ref('');
  const isSubmitting = ref(false);

  const formRef = ref();

  /** 获取下一个版本号 */
  async function fetchNextVersion(bumpType: BumpType) {
    if (!props.appId) return;
    const res = await HelmChartsService.getHelmChartSemver({
      appID: props.appId,
      bumpType,
    });
    nextVersion.value = res.next?.version || '';
  }

  /** 取消 */
  function handleCancel() {
    isShow.value = false;
  }

  /** 提交表单 */
  async function handleSubmit() {
    if (!formRef.value) return;

    try {
      await formRef.value.validate();
    } catch {
      return;
    }

    isSubmitting.value = true;
    try {
      const res = await HelmChartsService.createHelmChartBuild({
        appID: props.appId,
        bumpType: formData.value.bumpType,
        branch: formData.value.branch,
      });
      Message({ theme: 'success', message: t('创建构建成功') });
      emit('success', nextVersion.value, res.buildID || '');
      isShow.value = false;
    } catch {
      // API 调用失败，错误信息已由拦截器处理
    } finally {
      isSubmitting.value = false;
    }
  }

  /** 重置表单 */
  function resetForm() {
    formData.value = {
      branch: '',
      bumpType: 'patch',
    };
    nextVersion.value = '';
  }

  /** 监听版本类型变化 */
  watch(
    () => formData.value.bumpType,
    newVal => {
      // dialog 未打开时不发请求
      if (!isShow.value) return;
      fetchNextVersion(newVal);
    },
  );

  /** 监听 Dialog 打开，重置表单并获取下一个版本号 */
  watch(isShow, val => {
    if (!val) {
      resetForm();
    } else {
      nextTick(() => {
        formRef.value?.clearValidate?.();
      });
      // resetForm 将 bumpType 重置为 'patch'，若之前已是 'patch' 则 watcher 不会触发，需手动调用
      fetchNextVersion(formData.value.bumpType);
      formData.value.branch = appDetailStore?.appDetail?.helmSpec?.helmSource?.gitRepoConfig?.revision || '';
    }
  });
</script>
