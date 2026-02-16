package amp

import "github.com/lorenhsu1128/marcus-sidecar-on-windows/internal/adapter"

func init() {
	adapter.RegisterFactory(func() adapter.Adapter {
		return New()
	})
}
