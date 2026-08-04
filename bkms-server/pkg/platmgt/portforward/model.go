// Package portforward 提供 port-forward 白名单管理能力，作为 platmgt 模块的一部分。
package portforward

// WhitelistEntry 定义 port-forward 白名单中的一条记录。
// 每条记录代表一个被允许 port-forward 的非正式环境。
type WhitelistEntry struct {
	// EnvID 仅允许非正式环境 ID（同时作为文档 _id）。
	EnvID string `bson:"_id" json:"envID"`
}
