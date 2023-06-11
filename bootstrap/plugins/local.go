package plugins

import (
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
)

type LangGoLocal struct {
}

// Name .
func (lg *LangGoLocal) Name() string {
	return "Local"
}

// New .
func (lg *LangGoLocal) New() interface{} {
	return nil
}

// Health .
func (lg *LangGoLocal) Health() {}

// Close .
func (lg *LangGoLocal) Close() {}

// Flag .
func (lg *LangGoLocal) Flag() bool {
	return bootstrap.NewConfig("").Local.Enabled
}

func init() {
	p := &LangGoLocal{}
	RegisteredPlugin(p)
}
