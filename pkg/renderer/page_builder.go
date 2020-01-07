package renderer

import "github.com/ViBiOh/fibr/pkg/provider"

func (a app) newPageBuilder() *provider.PageBuilder {
	return (&provider.PageBuilder{}).Config(a.config)
}
