// Package lease 提供 Redis 可续租的分布式独占租约。
package lease

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	renewScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("pexpire", KEYS[1], ARGV[2])
end
return 0`)
	releaseScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
end
return 0`)
)

// Lease 表示仅持有者可续租和释放的 Redis 锁。
type Lease struct {
	client *redis.Client
	key    string
	token  string
	ttl    time.Duration // 锁过期时间
}

// Acquire 尝试获取租约；acquired=false 表示已有其他实例持有。
func Acquire(ctx context.Context, client *redis.Client, key string, ttl time.Duration) (*Lease, bool, error) {
	if client == nil || key == "" || ttl <= 0 {
		return nil, false, errors.New("invalid redis lease config")
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, false, err
	}
	token := hex.EncodeToString(raw)
	acquired, err := client.SetNX(ctx, key, token, ttl).Result()
	if err != nil || !acquired {
		return nil, acquired, err
	}
	return &Lease{client: client, key: key, token: token, ttl: ttl}, true, nil
}

// Renew 仅在当前实例仍持有租约时延长过期时间。
func (l *Lease) Renew(ctx context.Context) (bool, error) {
	if l == nil || l.client == nil {
		return false, errors.New("redis lease is nil")
	}
	result, err := renewScript.Run(ctx, l.client, []string{l.key}, l.token, l.ttl.Milliseconds()).Int64()
	return result == 1, err
}

// Release 仅删除当前实例持有的租约。
func (l *Lease) Release(ctx context.Context) error {
	if l == nil || l.client == nil {
		return nil
	}
	_, err := releaseScript.Run(ctx, l.client, []string{l.key}, l.token).Result()
	return err
}

// KeepAlive 在后台续租，返回停止函数。续租失败会取消 lostCtx。
func (l *Lease) KeepAlive(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	interval := l.ttl / 3
	if interval < time.Second {
		interval = time.Second
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ok, err := l.Renew(ctx)
				if err != nil || !ok {
					cancel()
					return
				}
			}
		}
	}()
	stop := func() {
		cancel()
		<-done
	}
	return ctx, stop
}
