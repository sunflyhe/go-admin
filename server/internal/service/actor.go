// 跨入口复用的最小业务对象:操作者与登录元数据。
// Handler 将 HTTP 语义(IP、User-Agent、登录态)转换成这些结构后传入 Service,
// 使 Service 可以被 CLI、定时任务、消息消费等非 HTTP 入口直接复用。
package service

// Actor 当前操作者的最小描述,由认证中间件从登录态转换而来。
type Actor struct {
	ID       int64
	Username string
	IsSuper  bool
}

// LoginMeta 登录场景的请求元数据,由 Handler 显式构造(仅登录日志需要)。
type LoginMeta struct {
	IP        string
	UserAgent string
}
