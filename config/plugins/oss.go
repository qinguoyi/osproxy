package plugins

// Oss 阿里云cos配置
type Oss struct {
	EndPoint        string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	AccessKeyId     string `mapstructure:"access_key_id" json:"access_key_id" yaml:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret" json:"access_key_secret" yaml:"access_key_secret"`
	Enabled         bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}
