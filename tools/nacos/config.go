package nacos

type Config struct {
	Group     string `toml:"group"`
	Endpoint  string `toml:"endpoint"`
	Namespace string `toml:"namespace"`
	AccessKey string `toml:"accessKey"`
	SecretKey string `toml:"secretKey"`
	OpenKMS   bool   `toml:"openKMS"`
	RegionId  string `toml:"regionId"`
}
