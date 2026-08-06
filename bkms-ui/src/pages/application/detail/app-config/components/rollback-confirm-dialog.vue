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
        <SvgIcon
          :height="42"
          icon="bkms-icon-tishi"
          :width="42"
        />
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ $t('确认回滚到') }} V{{ versionData?.version }}？
        </span>
      </div>
    </template>

    <div class="text-[14px] text-[#313238] mb-[16px] mt-[36px]">
      {{ $t('回滚后，该版本将成为新的当前版本。原配置会保留在历史记录中，您仍可再次切换。') }}
    </div>

    <!-- 版本信息 -->
    <div class="bg-[#F5F7FA] mb-[16px] py-[12px] px-[16px] text-[12px]">
      <ul class="list-disc pl-[20px] leading-[20px]">
        <li>
          <span class="text-[#4D4F56]">{{ $t('回滚到') }}：</span>
          <span class="text-[#313238]">V{{ versionData?.version }}</span>
        </li>
        <li>
          <span class="text-[#4D4F56]">{{ $t('版本时间') }}：</span>
          <span class="text-[#313238]">
            {{ versionData?.createdAt ? dayjs(versionData?.createdAt).format('YYYY-MM-DD HH:mm:ss') : '--' }}
          </span>
        </li>
        <li>
          <span class="text-[#4D4F56]">{{ $t('版本描述') }}：</span>
          <span class="text-[#313238]">{{ versionData?.description || '--' }}</span>
        </li>
      </ul>
    </div>

    <!-- 回滚备注 -->
    <Form
      ref="formRef"
      form-type="vertical"
      :model="formData"
      :rules="rules"
    >
      <Form.FormItem
        :label="$t('回滚备注')"
        property="remark"
        required
      >
        <Input
          v-model="formData.remark"
          :placeholder="$t('请输入回滚备注')"
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
          theme="danger"
          @click="handleConfirm"
        >
          {{ $t('确认回滚') }}
        </Button>
        <Button @click="handleHidden">
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { reactive, ref, watch } from 'vue';

  import { Button, Dialog, Form, Input, Message } from 'bkui-vue';
  import dayjs from 'dayjs';
  import { AppConfigFileVersionOutputObj } from '~/@types/v1/app-config-files';
  import { AppConfigFilesService } from '~/api/modules/v1';
  import { hasErrorCode } from '~/common/util';
  import SvgIcon from '~/components/svg-icon.vue';
  import { useAppDetail } from '~/stores/app-detail';

  interface Props {
    currentVersion?: number;
    versionData?: null | VersionItem;
  }

  /** 版本项类型 */
  interface VersionItem extends AppConfigFileVersionOutputObj {
    isCurrent: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    versionData: null,
  });

  const emit = defineEmits<{ (e: 'rollback-success'): void }>();

  const appDetailStore = useAppDetail();

  const isShow = defineModel<boolean>('isShow', { default: false });

  /** 表单数据 */
  const formData = reactive({
    remark: '',
  });

  /** 弹窗打开时默认填充回滚备注 */
  watch(isShow, val => {
    if (val && props.versionData?.version) {
      formData.remark = `回滚到 V${props.versionData.version}`;
    }
  });

  /** 表单校验规则 */
  const rules = {
    remark: [
      {
        message: '请输入回滚备注',
        required: true,
        trigger: 'blur',
        validator: (val: string) => val.trim().length > 0,
      },
    ],
  };

  /** 表单 ref */
  const formRef = ref<InstanceType<typeof Form> | null>(null);
  const loading = ref(false);

  /** 确认回滚 */
  async function handleConfirm() {
    const valid = await formRef.value
      ?.validate()
      .then(() => true)
      .catch(() => false);
    if (!valid) return;

    if (!props.versionData) return;

    try {
      loading.value = true;
      await AppConfigFilesService.rollbackAppConfigFileVersion(
        {
          appID: appDetailStore.appID,
          id: props.versionData?.id ?? '',
          description: formData.remark.trim(),
          currentVersion: props.currentVersion,
        },
        { interceptorErr: false },
      );
      handleHidden();
      emit('rollback-success');
    } catch (err) {
      if (hasErrorCode(err, 'APP_CONFIG_FILE_VERSION_CONFLICT')) {
        Message({
          theme: 'error',
          message: '当前配置已被他人更新。为避免数据被覆盖，请刷新页面获取最新版本后重新编辑。',
        });
      }
    } finally {
      loading.value = false;
    }
  }

  /** 关闭弹窗 */
  function handleHidden() {
    isShow.value = false;
    loading.value = false;
    formData.remark = '';
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
