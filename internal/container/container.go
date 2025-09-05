package container

import (
	"github.com/gocraft/dbr/v2"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/store"
)

// Container 依赖注入容器，管理所有服务依赖
type Container struct {
	session *dbr.Session

	// Stores
	userStore   store.UserStore
	policyStore store.PolicyStore

	// Services
	userService   *service.UserService
	policyService *service.PolicyService
}

// NewContainer 创建新的容器实例
// 参数:
//   - session: 数据库会话
// 返回值:
//   - *Container: 初始化完成的容器实例
func NewContainer(session *dbr.Session) *Container {
	c := &Container{
		session: session,
	}

	// 初始化存储层
	c.initStores()

	// 初始化服务层
	c.initServices()

	return c
}

// initStores 初始化存储层
func (c *Container) initStores() {
	c.userStore = store.NewUserStore(c.session)
	c.policyStore = store.NewPolicyStore(c.session)
}

// initServices 初始化服务层
func (c *Container) initServices() {
	c.userService = service.NewUserService(c.userStore, c.policyStore)
	c.policyService = service.NewPolicyService(c.policyStore)
}

// GetUserService 获取用户服务
// 返回值:
//   - *service.UserService: 用户服务实例
func (c *Container) GetUserService() *service.UserService {
	return c.userService
}

// GetPolicyService 获取策略服务
// 返回值:
//   - *service.PolicyService: 策略服务实例
func (c *Container) GetPolicyService() *service.PolicyService {
	return c.policyService
}

// Close 关闭容器，清理资源
// 返回值:
//   - error: 关闭过程中的错误
func (c *Container) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}