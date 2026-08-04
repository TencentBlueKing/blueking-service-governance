<template>
  <Popover
    :offset="{
      mainAxis: 6,
      crossAxis: -10,
    }"
    placement="bottom-start"
    theme="light"
    trigger="hover"
  >
    <i class="bkms-icon bkms-icon-warning-circle cursor-pointer text-[16px] text-[#979BA5]"></i>
    <template #content>
      <div class="w-[596px] pt-[8px]">
        <div class="mb-[12px] text-[14px] font-bold text-[#313238]">{{ $t('环境分类说明') }}</div>
        <Table
          class="w-full"
          :data="categoryRows"
          :row-class-name="getRowClassName"
          :row-height="56"
        >
          <TableColumn
            :label="$t('环境分类')"
            :width="260"
          >
            <template #default="{ row }: { row: CategoryRow }">
              <div
                v-if="row.type === 'nonProduction'"
                class="flex items-center gap-[12px]"
              >
                <Tag
                  class="min-w-[40px] !justify-center"
                  :class="envTypeTagStrokeClassMap.development"
                  type="stroke"
                >
                  {{ $t('开发') }}
                </Tag>
                <Tag
                  class="min-w-[40px] !justify-center"
                  :class="envTypeTagStrokeClassMap.test"
                  type="stroke"
                >
                  {{ $t('测试') }}
                </Tag>
                <Tag
                  class="min-w-[40px] !justify-center"
                  :class="envTypeTagStrokeClassMap.staging"
                  type="stroke"
                >
                  {{ $t('预发布') }}
                </Tag>
              </div>
              <Tag
                v-else
                class="min-w-[40px] !justify-center"
                :class="envTypeTagStrokeClassMap.production"
                type="stroke"
              >
                {{ $t('生产') }}
              </Tag>
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('开发模式')"
            :width="140"
          >
            <template #default="{ row }: { row: CategoryRow }">
              <div class="flex items-center">
                <Done
                  v-if="row.type === 'nonProduction'"
                  class="mr-[6px]"
                  :height="20"
                  text="#2CAF5E"
                  :width="20"
                />
                <Error
                  v-else
                  class="mr-[10px]"
                  :height="16"
                  text="#F59500"
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
                <i18n-t keypath="需{0}后才能部署">
                  <span class="text-[#F59500]">{{ $t('镜像晋级') }}</span>
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
  import { Table, TableColumn } from '@blueking/table';
  import { Popover, Tag } from 'bkui-vue';
  import { Done, Error } from 'bkui-vue/lib/icon';
  import { envTypeTagStrokeClassMap } from '~/composables/use-env-manager';

  interface CategoryRow {
    type: 'nonProduction' | 'production';
  }

  const categoryRows: CategoryRow[] = [{ type: 'nonProduction' }, { type: 'production' }];

  function getRowClassName({ row }: { row: CategoryRow }) {
    return row.type === 'production' ? '!bg-[#FFFBF5]' : '';
  }
</script>
