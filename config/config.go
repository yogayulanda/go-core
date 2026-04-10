package config

import "time"

type ValidationIssue struct {
	Section string
	Field   string
	Message string
}

type ValidationErrors struct {
	Issues []ValidationIssue
}

func (e *ValidationErrors) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return ""
	}

	out := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		out = append(out, issue.Message)
	}
	return joinValidationMessages(out)
}

type Config struct {
	App           AppConfig
	Databases     map[string]DBConfig
	Migration     MigrationConfig
	GRPC          GRPCConfig
	HTTP          HTTPConfig
	Observability ObservabilityConfig
	Auth          AuthConfig
	Redis         RedisConfig
	Memcached     MemcachedConfig
	Kafka         KafkaConfig
}

type AppConfig struct {
	ServiceName     string
	Environment     string
	LogLevel        string
	ShutdownTimeout time.Duration
}

type DBConfig struct {
	Driver          string
	DSN             string
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	Params          string
	Required        bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

type MigrationConfig struct {
	AutoRun     bool
	DBName      string
	Dir         string
	LockEnabled bool
	LockKey     string
	LockTimeout time.Duration
}

type GRPCConfig struct {
	Port        int
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string
}

type HTTPConfig struct {
	Port         int
	TLSEnabled   bool
	TLSCertFile  string
	TLSKeyFile   string
	PprofEnabled bool
}

type ObservabilityConfig struct {
	OTLPEndpoint       string
	OTLPInsecure       bool
	OTLPCACertFile     string
	TraceSamplingRatio float64
}

type AuthConfig struct {
	InternalJWT InternalJWTConfig
}

type InternalJWTConfig struct {
	Enabled        bool
	PublicKey      string
	Issuer         string
	Audience       string
	Leeway         time.Duration
	IncludeMethods []string
	ExcludeMethods []string
}

type RedisConfig struct {
	Enabled  bool
	Address  string
	Password string
	DB       int
}

type MemcachedConfig struct {
	Enabled bool
	Servers []string
	Timeout time.Duration
}

type KafkaConfig struct {
	Enabled  bool
	Brokers  []string
	ClientID string
}
