/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetLatestEnvReportRequest, ClusterReportOutput } from '~/@types/v1/bkintegrations-kubeinsight';

export const BkintegrationsKubeinsightService = {
  /**
   * 获取最新环境巡检报告
   *
   * @method GET
   * @path /kube-insight/reports
   * @tag bkintegrations-kubeinsight
   * @param envID query string required 环境 ID
   * @param generatePDF query boolean 是否生成 PDF
   * @response 200 GetLatestEnvReportOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getLatestEnvReport: async <Request extends GetLatestEnvReportRequest = GetLatestEnvReportRequest, ResponseData = ClusterReportOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/kube-insight/reports')(params, config),
};
