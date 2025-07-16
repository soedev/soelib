package specialdb

/**
  redis  数据库连接以及管理
*/

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	splunkredis "github.com/signalfx/splunk-otel-go/instrumentation/github.com/gomodule/redigo/splunkredigo/redis"
	"github.com/soedev/soelib/common/des"
	"time"
)

// RedisConfig 连接配置
type RedisConfig struct {
	Host        string
	Password    string
	MaxIdle     int   //最大空闲连接数
	MaxActive   int   //在给定时间内，允许分配的最大连接数（当为零时，没有限制）
	IdleTimeout int64 //在给定时间内将会保持空闲状态，若到达时间限制则关闭连接（当为零时，没有限制）
	Db          int   //设置redisDb
	EnableTrace bool
}

type RedisTemplate struct {
	Pool        *redis.Pool
	EnableTrace bool // 启用链路
}

// ConnRedis  设置redis 缓存
func ConnRedis(config RedisConfig) (*RedisTemplate, error) {
	// 解密密码
	if config.Password != "" {
		config.Password = des.DecryptDESECB([]byte(config.Password), des.DesKey)
	}
	pool := &redis.Pool{
		MaxIdle:     config.MaxIdle,
		MaxActive:   config.MaxActive,
		IdleTimeout: time.Duration(config.IdleTimeout),
		Dial: func() (redis.Conn, error) {
			db := redis.DialDatabase(config.Db)
			dbPwd := redis.DialPassword(config.Password)
			if config.EnableTrace {
				// 启用链路追踪
				c, err := splunkredis.Dial("tcp", config.Host, db, dbPwd)
				if err != nil {
					return nil, err
				}
				return c, err
			} else {
				c, err := redis.Dial("tcp", config.Host, db, dbPwd)
				if err != nil {
					return nil, err
				}
				return c, err
			}
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute*10 {
				// 空闲大于10分钟才测试
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
	return &RedisTemplate{
		Pool:        pool,
		EnableTrace: config.EnableTrace,
	}, nil
}

// GetAndRenew 读取数据，当数据存在时自动续约过期时间(秒)
func (s *RedisTemplate) GetAndRenew(key string, exTime int, ctx context.Context) (value []byte, err error) {
	conn := s.Pool.Get()
	defer safeClose(conn, "GetAndRenew")
	getArgs := []interface{}{key}
	if s.EnableTrace {
		getArgs = append(getArgs, ctx)
	}
	reply, err := redis.Bytes(conn.Do("GET", getArgs...))
	if err != nil {
		fmt.Printf("------------RedisTemplate: GetAndRenew GET:err:%v------------\n", err)
		return nil, err
	}

	expireArgs := []interface{}{key, exTime}
	if s.EnableTrace {
		expireArgs = append(expireArgs, ctx)
	}
	_, err = conn.Do("EXPIRE", expireArgs...)

	if err != nil {
		fmt.Printf("------------RedisTemplate: GetAndRenew EXPIRE:err:%v------------\n", err)
	}
	return reply, nil
}

// Renew  续期： key  续期时间（毫秒）
func (s *RedisTemplate) Renew(key string, exTime int, ctx context.Context) error {
	conn := s.Pool.Get()
	defer safeClose(conn, "Renew")
	expireArgs := []interface{}{key, exTime}
	if s.EnableTrace {
		expireArgs = append(expireArgs, ctx)
	}
	_, err := conn.Do("EXPIRE", expireArgs...)
	if err != nil {
		fmt.Printf("------------RedisTemplate: Renew EXPIRE:err:%v------------\n", err)
		return err
	}
	return nil
}

// SetString 缓存数据：key 数据 过期时间（毫秒）
func (s *RedisTemplate) SetString(key string, data string, times int, ctx context.Context) error {
	conn := s.Pool.Get()
	defer safeClose(conn, "SetString")
	var args []interface{}
	if times == -1 {
		args = []interface{}{key, data}
	} else {
		args = []interface{}{key, data, "EX", times}
	}
	if s.EnableTrace {
		args = append(args, ctx)
	}
	_, err := conn.Do("SET", args...)
	return err
}

// Set 缓存数据：key 数据 过期时间（毫秒）
func (s *RedisTemplate) Set(key string, data interface{}, time int, ctx context.Context) error {
	conn := s.Pool.Get()
	defer safeClose(conn, "Set")
	value, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var args []interface{}
	if time == -1 {
		args = []interface{}{key, value}
	} else {
		args = []interface{}{key, value, "EX", time}
	}
	if s.EnableTrace {
		args = append(args, ctx)
	}
	_, err = conn.Do("SET", args...)
	if err != nil {
		fmt.Printf("------------RedisTemplate: Set err:%v------------\n", err)
		return err
	}
	return nil
}

// Exists 检测 key 是否存在
func (s *RedisTemplate) Exists(key string, ctx context.Context) bool {
	conn := s.Pool.Get()
	defer safeClose(conn, "Exists")

	existsArgs := []interface{}{key}
	if s.EnableTrace {
		existsArgs = append(existsArgs, ctx)
	}

	exists, err := redis.Bool(conn.Do("EXISTS", existsArgs...))
	if err != nil {
		fmt.Printf("------------RedisTemplate: Exists err:%v------------\n", err)
		return false
	}
	return exists
}

// Get 读取缓存数据
func (s *RedisTemplate) Get(key string, ctx context.Context) ([]byte, error) {
	conn := s.Pool.Get()
	defer safeClose(conn, "Get")

	getArgs := []interface{}{key}
	if s.EnableTrace {
		getArgs = append(getArgs, ctx)
	}

	reply, err := redis.Bytes(conn.Do("GET", getArgs...))
	if err != nil {
		if !errors.Is(err, redis.ErrNil) {
			fmt.Printf("------------RedisTemplate: Get err:%v------------\n", err)
		}
		return nil, err
	}
	return reply, nil
}

// Delete 删除缓存数据
func (s *RedisTemplate) Delete(key string, ctx context.Context) (bool, error) {
	conn := s.Pool.Get()
	defer safeClose(conn, "Get")

	delArgs := []interface{}{key}
	if s.EnableTrace {
		delArgs = append(delArgs, ctx)
	}

	return redis.Bool(conn.Do("DEL", delArgs...))
}

// LikeDeletes  like 删除
func (s *RedisTemplate) LikeDeletes(key string, ctx context.Context) error {
	conn := s.Pool.Get()
	defer safeClose(conn, "LikeDeletes")

	keysArgs := []interface{}{"*" + key + "*"}
	if s.EnableTrace {
		keysArgs = append(keysArgs, ctx)
	}

	keys, err := redis.Strings(conn.Do("KEYS", keysArgs...))
	if err != nil {
		fmt.Printf("------------RedisTemplate: LikeDeletes KEYS:err:%v------------\n", err)
		return err
	}

	for _, key := range keys {
		_, err = s.Delete(key, ctx)
		if err != nil {
			fmt.Printf("------------RedisTemplate: LikeDeletes Delete:err:%v------------\n", err)
			return err
		}
	}
	return nil
}

// Lock 加锁
func (s *RedisTemplate) Lock(lock, value string, expire int, ctx context.Context) (ok bool, err error) {
	conn := s.Pool.Get()
	defer safeClose(conn, "Lock")

	setArgs := []interface{}{lock, value, "EX", expire, "NX"}
	if s.EnableTrace {
		setArgs = append(setArgs, ctx)
	}

	//设置锁key-value和过期时间
	_, err = redis.String(conn.Do("SET", setArgs...))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Unlock 通过 Lua 脚本保障原子性操作
func (s *RedisTemplate) Unlock(key, value string, ctx context.Context) error {
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	reply, err := s.EvalLuaScript(luaScript, []string{key}, []interface{}{value}, ctx)
	if err != nil {
		fmt.Printf("------------RedisTemplate: Unlock EVAL:err:%v------------\n", err)
	}
	// 返回 0 表示未删除
	deleted, _ := redis.Int(reply, nil)
	if deleted == 0 {
		return errors.New("非法用户，无法释放该锁")
	}
	return nil
}

//func (s *RedisTemplate) Unlock(key, value string, ctx context.Context) (err error) {
//	conn := s.Pool.Get()
//	defer safeClose(conn, "Unlock")
//
//	args := []interface{}{key}
//	if s.EnableTrace {
//		args = append(args, ctx)
//	}
//
//	//获取锁value
//	setValue, err := redis.String(conn.Do("GET", args...))
//	if err != nil {
//		fmt.Printf("------------RedisTemplate: Unlock GET:err:%v------------\n", err)
//		return
//	}
//	//判断锁是否属于该释放锁的线程
//	if setValue != value {
//		err = errors.New("非法用户，无法释放该锁")
//		return
//	}
//
//	//属于该用户，直接删除该key
//	_, err = conn.Do("DEL", args...)
//	if err != nil {
//		fmt.Printf("------------RedisTemplate: Unlock DEL:err:%v------------\n", err)
//		return
//	}
//	return
//}

// Incr 自增
func (s *RedisTemplate) Incr(key string, ctx context.Context) (result int, err error) {
	conn := s.Pool.Get()
	defer safeClose(conn, "Incr")

	incrArgs := []interface{}{key}
	if s.EnableTrace {
		incrArgs = append(incrArgs, ctx)
	}

	result, err = redis.Int(conn.Do("INCR", incrArgs...))
	if err != nil {
		fmt.Printf("------------RedisTemplate: Incr INCR:err:%v------------\n", err)
		return result, err
	}
	return result, nil
}

// HasGetAll 获取所有数据：key 数据 过期时间（毫秒）
func (s *RedisTemplate) HasGetAll(hasKey string, ctx context.Context) ([][]byte, error) {
	conn := s.Pool.Get()
	defer safeClose(conn, "HasGetAll")

	getAllArgs := []interface{}{hasKey}
	if s.EnableTrace {
		getAllArgs = append(getAllArgs, ctx)
	}

	reply, err := redis.Values(conn.Do("HGETALL", getAllArgs...))
	if err != nil {
		return nil, err
	}
	m, _ := redisHGETALLToMap(reply)
	return m, nil
}

// EvalLuaScript 通用 Lua 脚本执行器
func (s *RedisTemplate) EvalLuaScript(script string, keys []string, args []interface{}, ctx context.Context) (interface{}, error) {
	conn := s.Pool.Get()
	defer safeClose(conn, "EvalLuaScript")
	// 将 keys 和 args 打平到 []interface{}
	redisArgs := make([]interface{}, 0, 2+len(keys)+len(args))
	redisArgs = append(redisArgs, script)
	redisArgs = append(redisArgs, len(keys))
	for _, k := range keys {
		redisArgs = append(redisArgs, k)
	}
	redisArgs = append(redisArgs, args...)

	// 添加 ctx（用于 trace）
	if s.EnableTrace {
		redisArgs = append(redisArgs, ctx)
	}
	return conn.Do("EVAL", redisArgs...)
}

func redisHGETALLToMap(values []interface{}) ([][]byte, error) {
	var result [][]byte
	for i := 0; i < len(values); i += 2 {
		value, ok := values[i+1].([]byte)
		if !ok {
			return nil, fmt.Errorf("unexpected type for value, got type %T", values[i+1])
		}
		result = append(result, value)
	}
	return result, nil
}

func safeClose(conn redis.Conn, tag string) {
	err := conn.Close()
	if err != nil {
		fmt.Printf("------------RedisTemplate: %s redis.Conn-close:err:%v------------\n", tag, err)
	}
}
