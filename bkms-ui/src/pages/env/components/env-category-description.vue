<template>
  <Popover
    ref="categoryPopoverRef"
    :offset="{
      mainAxis: 6,
      crossAxis: -10,
    }"
    placement="bottom-start"
    theme="light no-border-popover"
    trigger="hover"
  >
    <Help
      :height="16"
      text="#979BA5"
      :width="16"
    />
    <template #content>
      <div class="w-[460px] pt-[8px]">
        <div class="mb-[12px] text-[14px] font-bold text-[#313238]">{{ $t('环境分类说明') }}</div>
        <Table
          class="w-full"
          :data="categoryRows"
          :row-height="56"
        >
          <TableColumn
            :label="$t('环境分类')"
            :width="180"
          >
            <template #default="{ row }: { row: CategoryRow }">
              <div
                v-if="row.type === 'nonProduction'"
                class="flex items-center gap-[4px]"
              >
                <Tag
                  class="min-w-[40px] !justify-center"
                  :class="envTypeTagClassMap.development"
                  type="stroke"
                >
                  {{ $t('开发') }}
                </Tag>
                <Tag
                  class="min-w-[40px] !justify-center"
                  :class="envTypeTagClassMap.test"
                  type="stroke"
                >
                  {{ $t('测试') }}
                </Tag>
                <Tag
                  class="min-w-[40px] !justify-center"
                  :class="envTypeTagClassMap.staging"
                  type="stroke"
                >
                  {{ $t('预发布') }}
                </Tag>
              </div>
              <Tag
                v-else
                class="min-w-[40px] !justify-center"
                :class="envTypeTagClassMap.production"
                type="stroke"
              >
                {{ $t('生产') }}
              </Tag>
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('开发模式')"
            :width="90"
          >
            <template #header>
              <Popover
                placement="top"
                theme="light"
                trigger="hover"
                :z-index="10000"
                @after-show="keepCategoryPopoverVisible"
                @content-mouseenter="keepCategoryPopoverVisible"
              >
                <span class="border-b border-dashed border-[#979BA5]">{{ $t('开发模式') }}</span>
                <template #content>
                  <div class="w-[320px] text-[#4D4F56]">
                    {{ $t('支持通过 bkms-cli 上传二进制的方式热更新服务，更新过程不会重启容器实例。') }}
                    <span
                      class="ml-[4px] cursor-pointer text-[#3A84FF]"
                      @click.stop="goToTrpcDevModeDoc"
                    >
                      {{ $t('查看详细文档') }}
                    </span>
                  </div>
                </template>
              </Popover>
            </template>
            <template #default="{ row }: { row: CategoryRow }">
              <div class="flex items-center">
                <Done
                  v-if="row.type === 'nonProduction'"
                  class="mr-[4px]"
                  :height="20"
                  text="#2CAF5E"
                  :width="20"
                />
                <Error
                  v-else
                  class="mr-[8px]"
                  :height="16"
                  text="#EA3636"
                  :width="16"
                />
                <span>{{ row.type === 'nonProduction' ? $t('允许') : $t('禁止') }}</span>
              </div>
            </template>
          </TableColumn>
          <TableColumn :label="$t('镜像与部署')">
            <template #default="{ row }: { row: CategoryRow }">
              <template v-if="row.type === 'nonProduction'">
                {{ $t('可直接部署') }}
              </template>
              <template v-else>
                <i18n-t keypath="需 {0} 后才能部署">
                  <span class="text-[#EA3636]">{{ $t('镜像晋级') }}</span>
                </i18n-t>
              </template>
            </template>
          </TableColumn>
        </Table>
      </div>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
  import { ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Popover, Tag } from 'bkui-vue';
  import { Done, Error, Help } from 'bkui-vue/lib/icon';
  import { DOC_LINKS } from '~/common/const';
  import { envTypeTagClassMap } from '~/composables/use-env-manager';

  interface CategoryRow {
    type: 'nonProduction' | 'production';
  }

  const categoryRows: CategoryRow[] = [{ type: 'nonProduction' }, { type: 'production' }];
  const categoryPopoverRef = ref<{ stopHide: () => void }>();

  function goToTrpcDevModeDoc() {
    window.open(`${import.meta.env.BK_DOC_URL}${DOC_LINKS.TRPC_DEV_MODE}`, '_blank');
  }

  function keepCategoryPopoverVisible() {
    categoryPopoverRef.value?.stopHide();
  }
</script>
