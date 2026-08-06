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
    :width="hasUsages ? 680 : 480"
    @closed="emit('closed')"
  >
    <template #header>
      <div class="flex flex-col items-center pt-[10px]">
        <SvgIcon
          :height="42"
          icon="bkms-icon-tishi"
          :width="42"
        />
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ t('确定删除镜像 Tag ?') }}
        </span>
      </div>
    </template>
    <div v-bkloading="{ loading: isLoadingUsages }">
      <div
        v-if="!hasUsages"
        class="bg-[#F5F7FA] mb-[16px] py-[12px] px-[16px] text-[14px] text-[#4D4F56]"
      >
        {{ t('应用最近的部署记录中未检测到该镜像 Tag 的使用，但是镜像 Tag 一旦删除后无法找回') }}
      </div>
      <div
        v-else
        class="bg-[#F5F7FA] mb-[16px] py-[12px] px-[16px] text-[14px] text-[#4D4F56] font-medium"
      >
        <i18n-t
          keypath="该镜像 Tag ({0}) 目前正被以下环境使用，{1}，请谨慎确认是否删除"
          tag="p"
        >
          <span>{{ row?.tag }}</span>
          <span class="text-[#EA3636]">{{
            $t('删除后可能会影响已部署的环境（如扩缩容、重启、回滚将无法拉取该镜像）')
          }}</span>
        </i18n-t>
      </div>
      <!-- 有占用时展示环境列表 -->
      <Table
        v-if="hasUsages"
        class="mb-[16px]"
        :data="imageUsages?.usages || []"
      >
        <TableColumn
          field="envName"
          :label="$t('环境')"
          :min-width="160"
        >
          <template #default="{ row: usageRow }">
            {{ getEnvDisplayName(usageRow.envName ?? '') }}
          </template>
        </TableColumn>
        <TableColumn
          field="status"
          :label="$t('部署状态')"
          :min-width="120"
        >
          <template #default="{ row: usageRow }">
            <StatusIcon
              :status="usageRow.status"
              :status-color-map="statusMaps.statusColorMap"
              :status-text-map="statusMaps.statusTextMap"
            />
          </template>
        </TableColumn>
      </Table>
      <!-- 确认输入 -->
      <i18n-t
        class="text-[12px] text-[#4D4F56]"
        keypath="该操作不可恢复，请输入镜像 Tag：{0} 进行确认"
      >
        <span
          v-bk-tooltips="$t('点击复制')"
          class="font-bold text-[#EA3636] cursor-pointer rounded-[2px] px-[4px] hover:bg-[#FFEBEB]"
          @click="copyText(row?.tag ?? '')"
        >
          {{ row?.tag }}
        </span>
      </i18n-t>
      <Input
        v-model="deleteConfirmInput"
        class="mt-[16px]"
        clearable
        :placeholder="$t('请输入镜像 Tag')"
      />
    </div>
    <template #footer>
      <div class="flex justify-center">
        <Button
          class="mr-[8px]"
          :disabled="deleteConfirmInput !== row?.tag"
          :loading="isDeleting"
          theme="danger"
          @click="handleConfirm"
        >
          {{ t('删除') }}
        </Button>
        <Button
          :disabled="isDeleting"
          @click="isShow = false"
        >
          {{ t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Dialog, Input, Message } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppImageOutputObj, ImageTagUsagesOutputObj } from '~/@types/v1/images';
  import { ImagesService } from '~/api/modules/v1';
  import { copyText } from '~/common/util';
  import StatusIcon from '~/components/status-icon.vue';
  import { useDeployStatusMap } from '~/composables/use-deploy-status';
  import { useErrorHandler } from '~/composables/use-error-handler';
  import { useAppDetail } from '~/stores/app-detail';

  import type { AppType } from '~/composables/app-type';

  interface ApiErrorResponse {
    status: number;
    error: Record<string, unknown> & {
      details: {
        code: string;
        extras: unknown;
        message: string;
        module: string;
        system: string;
      }[];
    };
  }

  interface IProps {
    imageUsages: ImageTagUsagesOutputObj | null;
    isLoadingUsages: boolean;
    row: AppImageOutputObj | null;
    getEnvDisplayName: (envName: string) => string;
  }

  const isShow = defineModel<boolean>('isShow');
  const props = defineProps<IProps>();

  const emit = defineEmits<{
    closed: [];
    permError: [];
    success: [];
  }>();

  const { t } = useI18n();
  const { handleError } = useErrorHandler();
  const appDetailStore = useAppDetail();
  const { getDeployStatusMaps } = useDeployStatusMap();

  const deleteConfirmInput = ref('');
  const isDeleting = ref(false);

  const hasUsages = computed(() => Boolean(props.imageUsages?.usages?.length));
  const statusMaps = computed(() => getDeployStatusMaps(appDetailStore.appType as AppType));

  // 每次打开弹窗时重置输入与提交状态
  watch(isShow, open => {
    if (open) {
      deleteConfirmInput.value = '';
      isDeleting.value = false;
    }
  });

  const NO_AUTH_DEL_TAG = 'IMAGE_REPOSITORY_AUTH_REQUIRED';

  /** 确认删除镜像 Tag */
  async function handleConfirm() {
    const row = props.row;
    if (!row || deleteConfirmInput.value !== row.tag) return;
    isDeleting.value = true;
    try {
      await ImagesService.deleteAppImage(
        {
          appID: appDetailStore.appID,
          tag: row.tag ?? '',
        },
        { interceptorErr: false, originalResponse: true, needStatus: true },
      );
      Message({ message: t('镜像 Tag 删除成功'), theme: 'success' });
      isShow.value = false;
      emit('success');
    } catch (err: unknown) {
      const errorResponse = err as ApiErrorResponse;
      const { error, status } = errorResponse;
      // 无权限删除时通知父组件弹出专属提示弹窗
      // error 可能为 undefined（如 500 且响应体无法解析），需用可选链避免崩溃
      if (status === 500 && error?.details?.[0]?.code === NO_AUTH_DEL_TAG) {
        emit('permError');
      } else {
        // 其他错误使用默认处理：将 details 归一为字符串以匹配 BackendError 类型
        handleError(
          { ...error, status, details: JSON.stringify(error?.details ?? '') } as Parameters<typeof handleError>[0],
          status,
        );
      }
    } finally {
      isDeleting.value = false;
    }
  }
</script>
