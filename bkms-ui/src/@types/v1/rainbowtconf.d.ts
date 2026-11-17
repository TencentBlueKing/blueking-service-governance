/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：rainbowtconf

export interface GetRainbowTConfRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type PutRainbowTConfRequest = PutRainbowTConfInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface RainbowTConfResponse {
  data?: RainbowTConfOutput;
}

export interface PutRainbowTConfInput {
  /**
   * 七彩石页面地址，仅支持 https
   */
  url: string;
}

export interface RainbowTConfOutput {
  /**
   * 最近修改时间，未配置时为 null
   */
  updatedAt?: string;
  /**
   * 最近修改人，未配置时为空
   */
  updater?: string;
  /**
   * 七彩石页面地址，未配置时为空
   */
  url?: string;
}
