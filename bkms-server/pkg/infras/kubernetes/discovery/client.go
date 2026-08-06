/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package discovery

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/TencentBlueKing/gopkg/mapx"
	openapiv2 "github.com/google/gnostic/openapiv2"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/openapi"
	"k8s.io/client-go/rest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/cache/redis"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

const (
	// ResCacheTTL 资源信息默认过期时间 14 天
	ResCacheTTL = 14 * 24 * 60 * 60

	// ResCacheKeyPrefix 集群资源信息 Redis 缓存键前缀
	ResCacheKeyPrefix = "osrcp"

	// CacheLockTTL 自动重置 ServerGroup 缓存时间间隔 10 分钟
	CacheLockTTL = 10 * 60
)

// RedisCacheClient 是基于 Redis 缓存的包含单个集群资源信息 k8s 客户端
type RedisCacheClient struct {
	delegate discovery.DiscoveryInterface
	// 集群 ID
	clusterID string

	// cache 即 redis 缓存
	cache *redis.Cache
	// isCacheValid 为 false 则缓存无效
	isCacheValid bool
	// mutex 锁保护 isCacheValid 字段
	mutex sync.RWMutex
}

// NewRedisCacheClient create RedisCacheClient
func NewRedisCacheClient(cfg *cluster.Config) (*RedisCacheClient, error) {
	delegate, err := discovery.NewDiscoveryClientForConfig(cfg.Rest)
	if err != nil {
		return nil, err
	}
	return &RedisCacheClient{
		delegate:     delegate,
		clusterID:    cfg.ClusterID,
		cache:        redis.NewCache(ResCacheKeyPrefix),
		isCacheValid: true,
	}, nil
}

// RESTClient ...
func (c *RedisCacheClient) RESTClient() rest.Interface {
	return c.delegate.RESTClient()
}

// ServerGroups 获取集群中的 Group，包含 versions, preferred 信息（支持 redis 缓存）
func (c *RedisCacheClient) ServerGroups() (*metav1.APIGroupList, error) {
	// 由于 CachedDiscoveryInterface.ServerGroups() 接口没有提供 context.Context 参数，因此本处使用 context.TODO()
	ctx := context.TODO()

	if cachedBytes, err := c.readCache(ctx, ""); err == nil {
		cachedGroups := &metav1.APIGroupList{}
		if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), cachedBytes, cachedGroups); err == nil {
			return cachedGroups, nil
		}
	}

	liveGroups, err := c.delegate.ServerGroups()
	if err != nil {
		log.WarnNoContextf("cluster: %s, skip caching discovery info due to %v", c.clusterID, err)
		return liveGroups, err
	}
	if liveGroups == nil || len(liveGroups.Groups) == 0 {
		log.WarnNoContextf("cluster: %s, skip caching discovery info, no groups found", c.clusterID)
		return liveGroups, err
	}
	if err = c.writeCache(ctx, "", liveGroups); err != nil {
		// Redis 缓存写失败应该有通知机制
		log.WarnNoContextf("cluster: %s, failed to write cache due to %v", c.clusterID, err)
	}
	return liveGroups, nil
}

// ServerResourcesForGroupVersion 获取指定 Group 与 Version 拥有的资源（支持 redis 缓存）
func (c *RedisCacheClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	// 由于 CachedDiscoveryInterface.ServerResourcesForGroupVersion()
	// 接口没有提供 context.Context 参数，因此本处使用 context.TODO()
	ctx := context.TODO()

	if cachedBytes, err := c.readCache(ctx, groupVersion); err == nil {
		cachedResources := &metav1.APIResourceList{}
		if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), cachedBytes, cachedResources); err == nil {
			return cachedResources, nil
		}
	}

	liveResources, err := c.delegate.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		log.WarnNoContextf("cluster: %s, skip caching %s discovery info due to %v", c.clusterID, groupVersion, err)
		return liveResources, err
	}
	if liveResources == nil || len(liveResources.APIResources) == 0 {
		log.WarnNoContextf("cluster: %s, skip caching %s discovery info, no res found", c.clusterID, groupVersion)
		return liveResources, err
	}
	if err = c.writeCache(ctx, groupVersion, liveResources); err != nil {
		// Redis 缓存写失败应该有通知机制
		log.WarnNoContextf("cluster: %s, failed to write cache due to %v", c.clusterID, err)
	}
	return liveResources, nil
}

// ServerGroupsAndResources 获取集群中所有资源 Groups 与 Versions
func (c *RedisCacheClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return discovery.ServerGroupsAndResources(c)
}

// ServerPreferredResources 获取集群资源 preferred 版本
func (c *RedisCacheClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	// 在我们的使用场景中，若某个 Group（如 v1beta1.metrics.k8s.io）异常，
	// 不应当影响在其他 Group 中寻找 Preferred 的资源，因此这里只记录 Warning 日志并忽略
	ret, err := discovery.ServerPreferredResources(c)
	if err != nil {
		log.WarnNoContextf("fetch some group's version resources in cluster %s failed: %v", c.clusterID, err)
	}
	return ret, nil
}

// ServerPreferredNamespacedResources 获取集群命名空间维度资源 preferred 版本
func (c *RedisCacheClient) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	// 在我们的使用场景中，若某个 Group（如 v1beta1.metrics.k8s.io）异常，
	// 不应当影响在其他 Group 中寻找 PreferredNamespaced 的资源，因此这里只记录 Warning 日志并忽略
	ret, err := discovery.ServerPreferredNamespacedResources(c)
	if err != nil {
		log.WarnNoContextf("fetch some group's version resources in cluster %s failed: %v", c.clusterID, err)
	}
	return ret, nil
}

// ServerVersion 获取集群 Server 版本（git version）
func (c *RedisCacheClient) ServerVersion() (*version.Info, error) {
	return c.delegate.ServerVersion()
}

// OpenAPISchema 获取集群支持的 Swagger API Schema
func (c *RedisCacheClient) OpenAPISchema() (*openapiv2.Document, error) {
	return c.delegate.OpenAPISchema()
}

// OpenAPIV3 获取集群支持的 Swagger API Schema
func (c *RedisCacheClient) OpenAPIV3() openapi.Client {
	return c.delegate.OpenAPIV3()
}

// WithLegacy 返回 DiscoveryInterface 的 Legacy 版本
func (c *RedisCacheClient) WithLegacy() discovery.DiscoveryInterface {
	return c.delegate.WithLegacy()
}

// Invalidate 使缓存失效
func (c *RedisCacheClient) Invalidate() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.isCacheValid = false
}

// Fresh 检查缓存状态
func (c *RedisCacheClient) Fresh() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.isCacheValid
}

// ClearCache 清理缓存内容 慎用！
func (c *RedisCacheClient) ClearCache(ctx context.Context) error {
	log.Warnf(ctx, "invalidate cluster %s discovery cache", c.clusterID)

	if err := c.checkCacheLock(ctx); err != nil {
		return err
	}

	var ret []byte
	allGroupCacheKey := genCacheKey(c.clusterID, "")
	if err := c.cache.Get(ctx, allGroupCacheKey, &ret); err != nil {
		// 如果集群没有对应缓存，是取不到数据的，也不需要清理，因此忽略异常
		log.Warnf(ctx, "failed to get all group cache: %v", err)
		return nil
	}
	var allGroup map[string]any
	if err := json.Unmarshal(ret, &allGroup); err != nil {
		return err
	}
	// 逐个遍历 AllGroup 中缓存的 GroupVersion 并删除
	for _, group := range mapx.GetList(allGroup, "groups") {
		for _, ver := range mapx.GetList(group.(map[string]any), "versions") {
			groupVersion := mapx.GetStr(ver.(map[string]any), "groupVersion")
			cacheKey := genCacheKey(c.clusterID, groupVersion)
			if err := c.cache.Delete(ctx, cacheKey); err != nil {
				log.Warnf(ctx, "delete cache key %s failed: %v, continue", cacheKey, err)
			}
		}
	}
	// 最后再删除 AllGroup 的缓存
	if err := c.cache.Delete(ctx, allGroupCacheKey); err != nil {
		return errors.Wrapf(err, "delete all groups cache")
	}

	return c.setCacheLock(ctx)
}

// getResWithGroupVersion 根据指定的 Group, Version 获取对应资源信息
func (c *RedisCacheClient) getResWithGroupVersion(kind, groupVersion string) (*schema.GroupVersionResource, error) {
	all, err := c.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		// 指定的 group/version 在集群中未注册时，apiserver 返回 NotFound，
		// 归一为 ErrKindNotFound（语义等价于目标资源类型不存在）；
		// 其它错误（集群不通、鉴权失败等）保持原样，避免误判为资源类型缺失。
		if k8serrors.IsNotFound(err) {
			return nil, errors.Wrapf(ErrKindNotFound, "group %s in cluster %s", groupVersion, c.clusterID)
		}
		return nil, err
	}

	// 逐个检查出第一个同名资源，作为结果返回
	gvr, err := filterResByKind(kind, []*metav1.APIResourceList{all})
	if err != nil {
		return nil, errors.Wrapf(err, "find %s in cluster %s, group %s", kind, c.clusterID, groupVersion)
	}

	return gvr, nil
}

// getPreferredResource 获取指定资源当前集群 Preferred 版本
func (c *RedisCacheClient) getPreferredResource(kind string) (*schema.GroupVersionResource, error) {
	all, err := c.ServerPreferredResources()
	if err != nil {
		return nil, err
	}
	// 逐个检查出第一个同名资源，作为 Preferred 结果返回
	gvr, err := filterResByKind(kind, all)
	if err != nil {
		return nil, errors.Wrapf(err, "find %s in cluster %s", kind, c.clusterID)
	}
	return gvr, nil
}

// readCache 读缓存逻辑
func (c *RedisCacheClient) readCache(ctx context.Context, groupVersion string) ([]byte, error) {
	if !c.Fresh() {
		return nil, errors.New("cache invalidated")
	}

	key := genCacheKey(c.clusterID, groupVersion)
	if !c.cache.Exists(ctx, key) {
		return nil, errors.Errorf("key %s cache not exists", key.Key())
	}

	var ret []byte
	err := c.cache.Get(ctx, key, &ret)
	return ret, err
}

// writeCache 写缓存逻辑
func (c *RedisCacheClient) writeCache(ctx context.Context, groupVersion string, obj runtime.Object) error {
	key := genCacheKey(c.clusterID, groupVersion)

	bytes, err := runtime.Encode(scheme.Codecs.LegacyCodec(), obj)
	if err != nil {
		return err
	}

	err = c.cache.Set(ctx, key, bytes, ResCacheTTL*time.Second)
	if err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.isCacheValid = true
	return nil
}

// checkCacheLock 检查缓存锁
func (c *RedisCacheClient) checkCacheLock(ctx context.Context) error {
	lockCacheKey := genLockKey(c.clusterID)
	if c.cache.Exists(ctx, lockCacheKey) {
		return errors.Errorf(
			"the interval is too short for reset cluster %s cache, please try again later", c.clusterID,
		)
	}
	return nil
}

// setCacheLock 设置缓存锁
func (c *RedisCacheClient) setCacheLock(ctx context.Context) error {
	lockCacheKey := genLockKey(c.clusterID)
	return c.cache.Set(ctx, lockCacheKey, "locked", CacheLockTTL*time.Second)
}

var _ discovery.CachedDiscoveryInterface = &RedisCacheClient{}
