package plugins

// Minio minio配置
type Minio struct {
	EndPoint        string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id" json:"access_key_id" yaml:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key" json:"secret_access_key" yaml:"secret_access_key"`
	UseSSL          bool   `mapstructure:"use_ssl" json:"use_ssl" yaml:"use_ssl"`
	Enabled         bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}
