package channel

// RegistryOption 配置 Provider 注册表。
type RegistryOption func(*Registry) error

// WithProviders 在构造注册表时批量注册 Provider。
func WithProviders(providers ...Provider) RegistryOption {
	return func(registry *Registry) error {
		for _, provider := range providers {
			if err := registry.Register(provider); err != nil {
				return err
			}
		}
		return nil
	}
}
