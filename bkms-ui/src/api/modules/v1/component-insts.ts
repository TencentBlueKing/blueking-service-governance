/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { PreviewComponentInstRequest, PreviewOutput } from '~/@types/v1/component-insts';

export const ComponentInstsService = {
  /**
   * 预览组件实例（试运行）
   *
   * @method POST
   * @path /component-insts/preview
   * @tag component-insts
   * @param body body PreviewComponentInstInput required 预览组件实例请求
   * @response 200 PreviewOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  previewComponentInst: async <Request extends PreviewComponentInstRequest = PreviewComponentInstRequest, ResponseData = PreviewOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/component-insts/preview')(params, config),
};
