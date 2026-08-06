/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListDeployableImageTagsRequest, PaginatedDeployableImageTagOutputObjs, ListAppImagesRequest, PaginatedAppImagesOutputObjs, RefreshAppImagesRequest, RefreshResultInfoOutputObj, DeleteAppImageRequest, ImageEmptyOutput, ListImageTagDeployRecordsRequest, PaginatedImageTagDeployRecordOutputObjs, PromoteAppImageRequest, ListAppImageUsagesRequest, ImageTagUsagesOutputObj, ListPlatformBuildImagesRequest, RuntimeImagesOutputObjs, ListPlatformBuildImageTagsRequest, PaginatedRuntimeImageTagOutputObjs } from '~/@types/v1/images';

export const ImagesService = {
  /**
   * 获取应用在指定环境下可部署的镜像 TAG 列表
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/deployable-image-tags
   * @tag images
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param keyword query string 搜索关键字（按 TAG 名称模糊搜索）
   * @param page query number required 分页参数：页码，从 1 开始
   * @param pageSize query number required 分页参数：每页数量，支持 5/10/20/50/100
   * @response 200 ListDeployableImageTagsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  listDeployableImageTags: async <Request extends ListDeployableImageTagsRequest = ListDeployableImageTagsRequest, ResponseData = PaginatedDeployableImageTagOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/deployable-image-tags')(params, config),
  /**
   * 获取应用镜像列表
   *
   * @method GET
   * @path /apps/{appID}/images
   * @tag images
   * @param appID path string required 应用 ID
   * @param keyword query string 搜索关键字
   * @param page query number required 分页参数：页码，从 1 开始
   * @param pageSize query number required 分页参数：每页数量，支持 5/10/20/50/100
   * @response 200 ListAppImagesOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  listAppImages: async <Request extends ListAppImagesRequest = ListAppImagesRequest, ResponseData = PaginatedAppImagesOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/images')(params, config),
  /**
   * 手动刷新应用镜像快照
   *
   * @method POST
   * @path /apps/{appID}/images/refresh
   * @tag images
   * @param appID path string required 应用 ID
   * @response 200 RefreshAppImagesOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  refreshAppImages: async <Request extends RefreshAppImagesRequest = RefreshAppImagesRequest, ResponseData = RefreshResultInfoOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/images/refresh')(params, config),
  /**
   * 删除应用镜像
   *
   * 直接删除远端镜像 tag，并清理本地快照与晋级记录；不依赖 usages 结果做拦截
   *
   * @method DELETE
   * @path /apps/{appID}/images/{tag}
   * @tag images
   * @param appID path string required 应用 ID
   * @param tag path string required 镜像标签
   * @response 200 ImageEmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  deleteAppImage: async <Request extends DeleteAppImageRequest = DeleteAppImageRequest, ResponseData = ImageEmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/images/{tag}')(params, config),
  /**
   * 获取指定镜像 Tag 的部署记录列表
   *
   * @method GET
   * @path /apps/{appID}/images/{tag}/deploy-records
   * @tag images
   * @param appID path string required 应用 ID
   * @param tag path string required 镜像标签
   * @param page query number required 分页参数：页码，从 1 开始
   * @param pageSize query number required 分页参数：每页数量，支持 5/10/20/50/100
   * @response 200 ListImageTagDeployRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  listImageTagDeployRecords: async <Request extends ListImageTagDeployRecordsRequest = ListImageTagDeployRecordsRequest, ResponseData = PaginatedImageTagDeployRecordOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/images/{tag}/deploy-records')(params, config),
  /**
   * 制品晋级
   *
   * @method PATCH
   * @path /apps/{appID}/images/{tag}/promote
   * @tag images
   * @param appID path string required 应用 ID
   * @param tag path string required 镜像标签
   * @response 200 ImageEmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  promoteAppImage: async <Request extends PromoteAppImageRequest = PromoteAppImageRequest, ResponseData = ImageEmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/apps/{appID}/images/{tag}/promote')(params, config),
  /**
   * 检查应用镜像使用情况
   *
   * 独立返回镜像 tag 在当前生效工作负载中的占用情况，供前端做删除前风险提示
   *
   * @method GET
   * @path /apps/{appID}/images/{tag}/usages
   * @tag images
   * @param appID path string required 应用 ID
   * @param tag path string required 镜像标签
   * @response 200 ListAppImageUsagesOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  listAppImageUsages: async <Request extends ListAppImageUsagesRequest = ListAppImageUsagesRequest, ResponseData = ImageTagUsagesOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/images/{tag}/usages')(params, config),
  /**
   * 获取平台通用构建镜像列表
   *
   * @method GET
   * @path /platform-build-images
   * @tag images
   * @param type query string required 镜像类型：builder / runner
   * @param keyword query string 搜索关键字
   * @response 200 ListRuntimeImagesOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  listPlatformBuildImages: async <Request extends ListPlatformBuildImagesRequest = ListPlatformBuildImagesRequest, ResponseData = RuntimeImagesOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/platform-build-images')(params, config),
  /**
   * 获取平台通用构建镜像可用 TAG 列表
   *
   * 从本地镜像快照读取指定平台通用构建镜像的 TAG；如果本地还没有快照，会异步触发一次初始化同步
   *
   * @method GET
   * @path /platform-build-images/{imageID}/tags
   * @tag images
   * @param imageID path string required 平台通用构建镜像记录 ID
   * @param keyword query string 搜索关键字
   * @param page query number required 分页参数：页码，从 1 开始
   * @param pageSize query number required 分页参数：每页数量，支持 5/10/20/50/100
   * @response 200 ListRuntimeImageTagsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  listPlatformBuildImageTags: async <Request extends ListPlatformBuildImageTagsRequest = ListPlatformBuildImageTagsRequest, ResponseData = PaginatedRuntimeImageTagOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/platform-build-images/{imageID}/tags')(params, config),
};
