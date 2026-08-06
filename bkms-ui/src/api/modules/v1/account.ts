/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetCurrentUserRequest, RefreshTokenRequest, CreateTokenRequest, ValidateTokenRequest, GetRoleRequest, RoleInfo } from '~/@types/v1/account';

export const AccountService = {
  /**
   * Get current user
   *
   * @method GET
   * @path /simple_account/info
   * @tag account
   */
  getCurrentUser: async <Request extends GetCurrentUserRequest = GetCurrentUserRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/simple_account/info')(params, config),
  /**
   * Refresh user access token
   *
   * @method GET
   * @path /user_token/refresh
   * @tag account
   */
  refreshToken: async <Request extends RefreshTokenRequest = RefreshTokenRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/user_token/refresh')(params, config),
  /**
   * Get current user access token
   *
   * @method GET
   * @path /user_token/token
   * @tag account
   */
  createToken: async <Request extends CreateTokenRequest = CreateTokenRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/user_token/token')(params, config),
  /**
   * Validate user access token
   *
   * @method GET
   * @path /user_token/validate
   * @tag account
   */
  validateToken: async <Request extends ValidateTokenRequest = ValidateTokenRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/user_token/validate')(params, config),
  /**
   * 查询当前用户的平台角色
   *
   * @method GET
   * @path /users/me/role
   * @tag account
   * @response 200 GetRoleResponse OK
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  getRole: async <Request extends GetRoleRequest = GetRoleRequest, ResponseData = RoleInfo>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/users/me/role')(params, config),
};
