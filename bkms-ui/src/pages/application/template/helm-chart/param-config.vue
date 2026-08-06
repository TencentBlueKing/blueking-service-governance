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
  <div class="flex justify-center max-w-[1400px] w-full">
    <Form
      ref="formRef"
      class="w-full p-[20px]"
      label-width="120"
      :model="formData"
      :rules="rules"
    >
      <ToggleCard :name="$t('基础信息')">
        <div class="px-[245px]">
          <Form.FormItem
            :label="$t('应用名称')"
            property="name"
            required
          >
            <Input
              v-model.trim="formData.name"
              clearable
              :placeholder="$t('请输入 1-20 个字符的小写字母、数字、中划线，以小写字母开头，提交后不可修改')"
            />
          </Form.FormItem>
          <Form.FormItem
            :label="$t('应用 ID')"
            property="workspace"
            required
          >
            <Input
              v-model="formData.id"
              disabled
              :placeholder="$t('自动生成')"
            />
          </Form.FormItem>
        </div>
      </ToggleCard>
      <ToggleCard
        class="mt-[16px]"
        :name="`Helm Chart ${$t('配置')}`"
      >
        <div class="px-[245px]">
          <HelmChartSourceForm
            ref="sourceFormRef"
            :initial-data="formData.helmSpec"
            validate-prefix="helmSpec"
          />
        </div>
      </ToggleCard>
      <ToggleCard
        class="mt-[16px]"
        :name="$t('构建配置')"
      >
        <div class="px-[245px]">
          <!-- 创建应用：不展示推荐版本号 -->
          <HelmChartBuildForm
            v-model="formData.buildConfig"
            :force-disable-code-repo="shouldForceDisableCodeRepo"
            :show-tag-config="false"
            validate-prefix="buildConfig"
          />
        </div>
      </ToggleCard>
    </Form>
  </div>
</template>
<script lang="ts" setup>
  import type { PropType } from 'vue';
  import { ref, watch } from 'vue';

  import { Form, Input } from 'bkui-vue';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { BKMS_REGEX } from '~/common/const';
  import { useAgonesFromRoute } from '~/composables/use-agones';

  import HelmChartBuildForm from './helm-chart-build-form.vue';
  import HelmChartSourceForm from './helm-chart-source-form.vue';

  import type { CreateAppRequest, GetAppIDAutoSuffixOutput } from '~/@types/v1/app';

  const props = defineProps({
    form: {
      type: Object as PropType<CreateAppRequest>,
      default: () => ({}),
    },
  });

  const route = useRoute(); // 获取路由信息

  const { t } = useI18n();

  // 使用 Agones Hook 统一判断是否为 Agones 应用
  const { shouldForceDisableCodeRepo } = useAgonesFromRoute();

  const formData = ref(props.form);
  const formRef = ref<InstanceType<typeof Form> | null>(null);
  const sourceFormRef = ref<InstanceType<typeof HelmChartSourceForm> | null>(null);

  const rules = ref({
    name: [
      {
        required: true,
        message: t('必填项'),
        trigger: 'blur',
      },
      {
        message: t('请输入 1-20 个字符的小写字母、数字、中划线，以小写字母开头'),
        trigger: 'blur',
        validator: () => BKMS_REGEX.appNameRegex.test(formData.value.name || ''),
      },
    ],
    workspace: [
      {
        required: true,
        validator: () => !!route.params.space,
      },
    ],
    'buildConfig.imageBuildConfig.username': [
      {
        message: t('请输入镜像凭证'),
        trigger: 'blur',
        validator: () => {
          const buildConfig = formData.value.buildConfig;
          if (!buildConfig) return true; // 如果buildConfig不存在，跳过验证

          const username = !!buildConfig.imageBuildConfig?.username;
          const password = !!buildConfig.imageBuildConfig?.password;
          return (username && password) || (!username && !password);
        },
      },
    ],
  });

  function getValue() {
    const result = cloneDeep(formData.value);
    const sourceData = sourceFormRef.value?.getValue();
    if (sourceData) {
      result.helmSpec = sourceData;
    }
    return result;
  }

  function validate() {
    return formRef.value?.validate();
  }

  const appIdSuffix = ref('');
  // 获取应用 ID 后缀
  async function getAppIDAutoSuffix() {
    const ret = (await ApiServerService.GetAppIDAutoSuffix({}, { needRes: true }).catch(() => ({
      suffix: '',
    }))) as GetAppIDAutoSuffixOutput;
    appIdSuffix.value = ret.suffix ?? '';
  }
  getAppIDAutoSuffix();

  watch(
    () => formData.value.name,
    newVal => {
      if (!newVal) {
        formData.value.id = '';
        return;
      }
      formData.value.id = `${newVal}${appIdSuffix.value}`;
    },
    { immediate: true },
  );

  defineExpose({
    validate,
    getValue,
    getAppIDAutoSuffix,
  });
</script>
