package plugins

// Cos 腾讯云cos配置
type Cos struct {
	Appid     string `mapstructure:"appid" json:"appid" yaml:"appid"`
	Region    string `mapstructure:"region" json:"region" yaml:"region"`
	SecretId  string `mapstructure:"secret_id" json:"secret_id" yaml:"secret_id"`
	SecretKey string `mapstructure:"secret_key" json:"secret_key" yaml:"secret_key"`
	Enabled   bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}
