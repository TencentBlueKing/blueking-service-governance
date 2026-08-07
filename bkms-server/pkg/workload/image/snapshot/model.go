package snapshot

import "time"

// Image 镜像快照记录
type Image struct {
	// RepoKey 仓库实例唯一标识
	RepoKey string `bson:"repoKey"`
	// Tag 镜像标签名
	Tag string `bson:"tag"`
	// Digest 镜像摘要
	Digest string `bson:"digest,omitempty"`
	// Size 镜像大小（字节）
	Size int64 `bson:"size,omitempty"`
	// BuiltAt 镜像构建时间（可为空，由 detail syncer 补全）
	BuiltAt *time.Time `bson:"builtAt,omitempty"`
	// DetailSyncPending 标记该标签的详情需要重新拉取（如标签被同名构建覆盖），
	// 详情同步成功后清除
	DetailSyncPending bool `bson:"detailSyncPending,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// RepoSnapshotStatus 仓库快照状态
type RepoSnapshotStatus struct {
	// RepoKey 仓库实例唯一标识
	RepoKey string `bson:"repoKey"`
	// RepoName 仓库名称（用于前端展示）
	RepoName string `bson:"repoName"`
	// RefreshStatus 当前刷新状态
	RefreshStatus RefreshStatus `bson:"refreshStatus"`
	// LastRefreshedAt 最后成功刷新时间
	LastRefreshedAt *time.Time `bson:"lastRefreshedAt,omitempty"`
	// LastDetailSyncedAt 最后成功详情同步时间
	LastDetailSyncedAt *time.Time `bson:"lastDetailSyncedAt,omitempty"`
	// LastError 最后失败信息
	LastError string `bson:"lastError,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
