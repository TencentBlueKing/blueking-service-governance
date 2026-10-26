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
    :quick-close="false"
    :title="$t('修改镜像仓库')"
    :width="640"
    @closed="handleClose"
  >
    <Alert
      class="mb-[16px]"
      theme="warning"
    >
      <template #title>
        <ul class="list-disc pl-[16px] leading-[20px]">
          <li>{{ $t('修改后，新构建的镜像将推送到新仓库') }}</li>
          <li>{{ $t('存量镜像不会迁移，使用旧镜像的应用仍需重新构建后再部署') }}</li>
          <li>{{ $t('当前运行中的实例不受此影响') }}</li>
        </ul>
      </template>
    </Alert>

    <Form
      ref="formRef"
      form-type="vertical"
      :model="formData"
      :rules="formRules"
    >
      <Form.FormItem
        :label="$t('仓库类型')"
        required
      >
        <Radio.Group v-model="formData.type">
          <Radio label="builtin">{{ $t('平台镜像仓库') }}</Radio>
          <Radio label="external">{{ $t('外部镜像仓库') }}</Radio>
        </Radio.Group>
      </Form.FormItem>
      <template v-if="formData.type === 'external'">
        <Form.FormItem
          :label="$t('镜像仓库地址')"
          property="registry"
          required
        >
          <Input
            v-model.trim="formData.registry"
            :placeholder="$t('请输入镜像仓库地址，如：{0}', ['mirrors.tencent.com/my-namespace'])"
          />
          <div class="text-[#979BA5] text-[12px] leading-[20px] mt-[4px]">
            {{ $t('填写到命名空间即可，不带协议前缀，不含镜像名与 tag') }}
          </div>
        </Form.FormItem>
        <Form.FormItem
          :label="$t('镜像账号')"
          property="username"
          required
        >
          <Input
            v-model.trim="formData.username"
            :placeholder="$t('请输入账号')"
          />
        </Form.FormItem>
        <Form.FormItem :label="$t('镜像密码')">
          <!-- 密码首尾空格也是凭证内容，不能 trim -->
          <Input
            v-model="formData.password"
            :placeholder="$t('留空则沿用该空间已有外部仓库的密码')"
            type="password"
          />
        </Form.FormItem>
      </template>
    </Form>

    <Checkbox v-model="confirmed">
      {{ $t('我已确认目标仓库可用，并知晓存量镜像不会迁移') }}
    </Checkbox>

    <template #footer>
      <span
        v-bk-tooltips="{
          content: $t('请先勾选确认'),
          disabled: confirmed,
        }"
      >
        <Button
          class="mr-[8px]"
          :disabled="!confirmed"
          :loading="isSubmitting"
          theme="primary"
          @click="handleSubmit"
        >
          {{ $t('保存') }}
        </Button>
      </span>
      <Button @click="handleClose">
        {{ $t('取消') }}
      </Button>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed, reactive, ref, watch } from 'vue';

  import { Alert, Button, Checkbox, Dialog, Form, Input, Message, Radio } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { PlatmgtService } from '~/api/modules/v1';

  import type { ImageRegistryOutput, UpdateImageRegistryInput } from '~/@types/v1/platmgt';

  interface Props {
    /** 空间当前生效的镜像仓库，为空表示尚未加载或配置缺失 */
    registry?: ImageRegistryOutput | null;
    workspaceId: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    registry: null,
  });

  const emit = defineEmits<{ saved: [registry: ImageRegistryOutput] }>();

  const { t } = useI18n();

  const isShow = defineModel<boolean>('isShow');

  const formRef = ref<InstanceType<typeof Form>>();
  const formData = reactive<Required<UpdateImageRegistryInput>>({
    type: 'builtin',
    registry: '',
    username: '',
    password: '',
  });
  const confirmed = ref(false);
  const isSubmitting = ref(false);

  // 地址格式由后端校验，这里只做必填；密码留空由后端沿用已有外部仓库的密码，没有可沿用的才报错
  const formRules = computed(() => ({
    registry: [
      {
        required: true,
        message: t('请输入镜像仓库地址，如：{0}', ['mirrors.tencent.com/my-namespace']),
        trigger: 'blur',
      },
    ],
    username: [{ required: true, message: t('请输入账号'), trigger: 'blur' }],
  }));

  // 每次打开都按当前生效配置重置表单：当前是外部仓库时预填地址和账号，密码不回填，需要修改时再输入
  watch(isShow, visible => {
    if (!visible) return;

    const isExternal = props.registry?.type === 'external';
    formData.type = isExternal ? 'external' : 'builtin';
    formData.registry = isExternal ? props.registry?.registry || '' : '';
    formData.username = isExternal ? props.registry?.username || '' : '';
    formData.password = '';
    confirmed.value = false;
    formRef.value?.clearValidate();
  });

  function handleClose() {
    isShow.value = false;
  }

  async function handleSubmit() {
    if (!confirmed.value || isSubmitting.value) return;

    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;

    isSubmitting.value = true;
    try {
      const saved = await PlatmgtService.updatePlatWorkspaceImageRegistry(
        formData.type === 'builtin'
          ? { workspaceID: props.workspaceId, type: 'builtin' }
          : { workspaceID: props.workspaceId, ...formData },
        { validateCode: false },
      );
      Message({ message: t('修改成功'), theme: 'success' });
      emit('saved', saved);
      handleClose();
    } catch (err) {
      // 接口错误已由请求拦截器弹出后端返回的英文提示
      console.error(err);
    } finally {
      isSubmitting.value = false;
    }
  }
</script>
