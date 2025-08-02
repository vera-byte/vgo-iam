package config

// Config 应用配置
type AppConfig struct {
	GRPC struct {
		Port string `yaml:"port"`
	} `yaml:"grpc"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	Security struct {
		MasterKey string `yaml:"master_key"`
	} `yaml:"security"`
	Log LogConfig `yaml:"log"`
}
type LogConfig struct {
	Level     string `yaml:"level"`     // 日志级别: debug/info/warn/error
	Format    string `yaml:"format"`    // 日志格式: json/console
	Directory string `yaml:"directory"` // 日志文件目录
	Filename  string `yaml:"filename"`  // 日志文件名
	ToStdout  bool   `yaml:"to_stdout"` // 是否输出到终端
}
