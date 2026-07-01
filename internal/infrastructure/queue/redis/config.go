package redis

import "github.com/hibiken/asynq"

type Config struct {
	Addr     string
	Username string
	Password string
	DB       int
}

func ClientOpt(cfg Config) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
}
