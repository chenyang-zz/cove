package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boxify/api-go/internal/config"
)

func TestLoadFileUsesDefaultsWhenYAMLIsMissing(t *testing.T) {
	cfg, err := config.LoadFile(filepath.Join(t.TempDir(), "missing.yml"))
	if err != nil {
		t.Fatalf("LoadFile error = %v", err)
	}

	if cfg.App.Env != "development" {
		t.Fatalf("app env = %q", cfg.App.Env)
	}
	if cfg.HTTPAddr() != "0.0.0.0:8000" {
		t.Fatalf("http addr = %q", cfg.HTTPAddr())
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Fatalf("redis addr = %q", cfg.Redis.Addr)
	}
}

func TestLoadFileReadsNestedYAML(t *testing.T) {
	path := writeConfig(t, `
app:
  env: production
http:
  host: 127.0.0.1
  port: 9000
database:
  url: postgres://yaml
redis:
  addr: redis:6379
  username: redis-user
  password: redis-password
  db: 2
elasticsearch:
  url: http://es:9200
  username: es-user
  password: es-password
  api_key: es-api-key
neo4j:
  uri: bolt://neo4j:7687
  username: neo4j
  password: yaml-password
  database: yaml-db
jwt:
  secret: yaml-secret
  access_token_ttl: 2h
secret_key: 12345678901234567890123456789012
storage:
  backend: cos
  dir: /data/storage
  cos:
    bucket_url: https://bucket.cos.ap-guangzhou.myqcloud.com
    secret_id: yaml-secret-id
    secret_key: yaml-secret-key
    base_url: https://cdn.example.com
llm:
  provider: openai
  model: gpt-4o-mini
  embedding_model: text-embedding-3-small
  base_url: https://api.openai.com/v1
  api_key: sk-yaml
`)

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile error = %v", err)
	}

	if cfg.App.Env != "production" || cfg.HTTPAddr() != "127.0.0.1:9000" {
		t.Fatalf("cfg app/http = %#v", cfg)
	}
	if cfg.Database.URL != "postgres://yaml" || cfg.Redis.Addr != "redis:6379" {
		t.Fatalf("cfg database/redis = %#v", cfg)
	}
	if cfg.Redis.Username != "redis-user" || cfg.Redis.Password != "redis-password" || cfg.Redis.DB != 2 {
		t.Fatalf("cfg redis detail = %#v", cfg.Redis)
	}
	if cfg.Elasticsearch.URL != "http://es:9200" || cfg.Elasticsearch.Username != "es-user" || cfg.Elasticsearch.Password != "es-password" || cfg.Elasticsearch.APIKey != "es-api-key" {
		t.Fatalf("cfg elasticsearch = %#v", cfg.Elasticsearch)
	}
	if cfg.Storage.Backend != "cos" || cfg.Storage.Dir != "/data/storage" || cfg.Storage.COS.BucketURL != "https://bucket.cos.ap-guangzhou.myqcloud.com" || cfg.Storage.COS.SecretID != "yaml-secret-id" || cfg.Storage.COS.SecretKey != "yaml-secret-key" || cfg.Storage.COS.BaseURL != "https://cdn.example.com" {
		t.Fatalf("cfg storage = %#v", cfg.Storage)
	}
	if cfg.Neo4j.URI != "bolt://neo4j:7687" || cfg.Neo4j.Username != "neo4j" || cfg.Neo4j.Password != "yaml-password" || cfg.Neo4j.Database != "yaml-db" {
		t.Fatalf("cfg neo4j = %#v", cfg.Neo4j)
	}
	if cfg.LLM.Model != "gpt-4o-mini" || cfg.LLM.APIKey != "sk-yaml" {
		t.Fatalf("cfg llm = %#v", cfg.LLM)
	}
	if cfg.JWT.Secret != "yaml-secret" || cfg.JWT.AccessTokenTTL != "2h" {
		t.Fatalf("cfg jwt = %#v", cfg.JWT)
	}
}

func TestLoadFileEnvOverridesYAML(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_HOST", "localhost")
	t.Setenv("APP_PORT", "9100")
	t.Setenv("DATABASE_URL", "postgres://env")
	t.Setenv("REDIS_ADDR", "env-redis:6379")
	t.Setenv("REDIS_USERNAME", "env-redis-user")
	t.Setenv("REDIS_PASSWORD", "env-redis-password")
	t.Setenv("REDIS_DB", "3")
	t.Setenv("ES_HOST", "http://env-es:9200")
	t.Setenv("ES_USERNAME", "env-es-user")
	t.Setenv("ES_PASSWORD", "env-es-password")
	t.Setenv("ES_API_KEY", "env-es-api-key")
	t.Setenv("NEO4J_URI", "bolt://env-neo4j:7687")
	t.Setenv("NEO4J_USERNAME", "env-user")
	t.Setenv("NEO4J_PASSWORD", "env-password")
	t.Setenv("NEO4J_DATABASE", "env-db")
	t.Setenv("JWT_SECRET", "env-jwt")
	t.Setenv("JWT_ACCESS_TOKEN_TTL", "30m")
	t.Setenv("SECRET_KEY", "abcdefghijklmnopqrstuvwxyz123456")
	t.Setenv("STORAGE_BACKEND", "env-storage")
	t.Setenv("STORAGE_DIR", "/env/storage")
	t.Setenv("COS_BUCKET_URL", "https://env-bucket.cos.ap-guangzhou.myqcloud.com")
	t.Setenv("COS_SECRET_ID", "env-cos-secret-id")
	t.Setenv("COS_SECRET_KEY", "env-cos-secret-key")
	t.Setenv("COS_BASE_URL", "https://env-cdn.example.com")
	t.Setenv("LLM_PROVIDER", "deepseek")
	t.Setenv("LLM_MODEL", "deepseek-chat")
	t.Setenv("LLM_EMBEDDING_MODEL", "deepseek-embed")
	t.Setenv("LLM_BASE_URL", "https://env.example/v1")
	t.Setenv("LLM_API_KEY", "sk-env")

	path := writeConfig(t, `
app:
  env: production
http:
  host: 0.0.0.0
  port: 8001
database:
  url: postgres://yaml
redis:
  addr: yaml-redis:6379
neo4j:
  uri: bolt://yaml-neo4j:7687
  username: yaml-user
  password: yaml-password
  database: yaml-db
jwt:
  secret: yaml-jwt
llm:
  provider: openai
  model: yaml-model
`)

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile error = %v", err)
	}

	if cfg.App.Env != "test" || cfg.HTTPAddr() != "localhost:9100" {
		t.Fatalf("env override app/http failed: %#v", cfg)
	}
	if cfg.Database.URL != "postgres://env" || cfg.Redis.Addr != "env-redis:6379" {
		t.Fatalf("env override database/redis failed: %#v", cfg)
	}
	if cfg.Redis.Username != "env-redis-user" || cfg.Redis.Password != "env-redis-password" || cfg.Redis.DB != 3 {
		t.Fatalf("env override redis detail failed: %#v", cfg.Redis)
	}
	if cfg.Elasticsearch.URL != "http://env-es:9200" || cfg.Elasticsearch.Username != "env-es-user" || cfg.Elasticsearch.Password != "env-es-password" || cfg.Elasticsearch.APIKey != "env-es-api-key" {
		t.Fatalf("env override elasticsearch failed: %#v", cfg.Elasticsearch)
	}
	if cfg.Storage.COS.BucketURL != "https://env-bucket.cos.ap-guangzhou.myqcloud.com" || cfg.Storage.COS.SecretID != "env-cos-secret-id" || cfg.Storage.COS.SecretKey != "env-cos-secret-key" || cfg.Storage.COS.BaseURL != "https://env-cdn.example.com" {
		t.Fatalf("env override cos failed: %#v", cfg.Storage)
	}
	if cfg.Neo4j.URI != "bolt://env-neo4j:7687" || cfg.Neo4j.Username != "env-user" || cfg.Neo4j.Password != "env-password" || cfg.Neo4j.Database != "env-db" {
		t.Fatalf("env override neo4j failed: %#v", cfg.Neo4j)
	}
	if cfg.JWT.Secret != "env-jwt" || cfg.JWT.AccessTokenTTL != "30m" || cfg.LLM.APIKey != "sk-env" {
		t.Fatalf("env override secrets failed: %#v", cfg)
	}
}

func TestLoadFileReturnsErrorForInvalidYAML(t *testing.T) {
	path := writeConfig(t, "app:\n  env: [broken\n")

	if _, err := config.LoadFile(path); err == nil {
		t.Fatal("LoadFile error = nil, want invalid YAML error")
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
