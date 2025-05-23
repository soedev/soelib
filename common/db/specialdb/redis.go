package specialdb

/**
  redis  数据库连接以及管理
*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

// RedisConfig 连接配置
type RedisConfig struct {
	Host        string
	Password    string
	MaxIdle     int   //最大空闲连接数
	MaxActive   int   //在给定时间内，允许分配的最大连接数（当为零时，没有限制）
	IdleTimeout int64 //在给定时间内将会保持空闲状态，若到达时间限制则关闭连接（当为零时，没有限制）
	Db          int   //设置redisDb
}

type RedisTemplate struct {
	Pool *redis.Pool
}

var Redis *RedisTemplate

// ConnRedis  设置redis 缓存
func ConnRedis(config RedisConfig) error {
	pool := &redis.Pool{
		MaxIdle:     config.MaxIdle,
		MaxActive:   config.MaxActive,
		IdleTimeout: time.Duration(config.IdleTimeout),
		Dial: func() (redis.Conn, error) {
			db := redis.DialDatabase(config.Db)
			dbPwd := redis.DialPassword(config.Password)
			c, err := redis.Dial("tcp", config.Host, db, dbPwd)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	Redis = &RedisTemplate{
		Pool: pool,
	}
	return nil
}

// GetAndRenew 读取数据，当数据存在时自动续约过期时间(毫秒)
func (s *RedisTemplate) GetAndRenew(key string, exTime int) (value []byte, err error) {
	conn := s.Pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: GetAndRenew redis.Conn-close:err:%v------------", err)
		}
	}(conn)
	reply, err := redis.Bytes(conn.Do("GET", key))
	if err != nil {
		fmt.Printf("------------RedisTemplate: GetAndRenew GET:err:%v------------", err)
		return nil, err
	}
	_, err = conn.Do("EXPIRE", key, exTime)
	if err != nil {
		fmt.Printf("------------RedisTemplate: GetAndRenew EXPIRE:err:%v------------", err)
	}
	return reply, nil
}

// Renew  续期： key  续期时间（毫秒）
func (s *RedisTemplate) Renew(key string, exTime int) error {
	conn := s.Pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: Renew redis.Conn-close:err:%v------------", err)
		}
	}(conn)
	_, err := conn.Do("EXPIRE", key, exTime)
	if err != nil {
		fmt.Printf("------------RedisTemplate: Renew EXPIRE:err:%v------------", err)
		return err
	}
	return nil
}

// Set 缓存数据：key 数据 过期时间（毫秒）
func (s *RedisTemplate) Set(key string, data interface{}, time int) error {
	conn := s.Pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: Set redis.Conn-close:err:%v------------", err)
		}
	}(conn)
	value, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if time == -1 {
		_, err = conn.Do("SET", key, value)
	} else {
		_, err = conn.Do("SET", key, value, "EX", time)
	}
	if err != nil {
		fmt.Printf("------------RedisTemplate: Set err:%v------------", err)
		return err
	}
	return nil
}

// Exists 检测 key 是否存在
func (s *RedisTemplate) Exists(key string) bool {
	conn := s.Pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: Exists redis.Conn-close:err:%v------------", err)
		}
	}(conn)
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		fmt.Printf("------------RedisTemplate: Exists err:%v------------", err)
		return false
	}
	return exists
}

// Get 读取缓存数据
func (s *RedisTemplate) Get(key string) ([]byte, error) {
	conn := s.Pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: Get redis.Conn-close:err:%v------------", err)
		}
	}(conn)
	reply, err := redis.Bytes(conn.Do("GET", key))
	if err != nil {
		if !errors.Is(err, redis.ErrNil) {
			fmt.Printf("------------RedisTemplate: Get err:%v------------", err)
		}
		return nil, err
	}
	return reply, nil
}

// Delete 删除缓存数据
func (s *RedisTemplate) Delete(key string) (bool, error) {
	conn := s.Pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: Delete redis.Conn-close:err:%v------------", err)
		}
	}(conn)

	return redis.Bool(conn.Do("DEL", key))
}

// LikeDeletes  like 删除
func (s *RedisTemplate) LikeDeletes(key string) error {
	conn := s.Pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: LikeDeletes redis.Conn-close:err:%v------------", err)
		}
	}(conn)
	keys, err := redis.Strings(conn.Do("KEYS", "*"+key+"*"))
	if err != nil {
		fmt.Printf("------------RedisTemplate: LikeDeletes KEYS:err:%v------------", err)
		return err
	}
	for _, key := range keys {
		_, err = s.Delete(key)
		if err != nil {
			fmt.Printf("------------RedisTemplate: LikeDeletes Delete:err:%v------------", err)
			return err
		}
	}
	return nil
}

// Lock 加锁
func (s *RedisTemplate) Lock(lock, value string, expire int) (ok bool, err error) {
	c := s.Pool.Get()
	defer func(c redis.Conn) {
		err := c.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: Lock redis.Conn-close:err:%v------------", err)
		}
	}(c)
	//设置锁key-value和过期时间
	_, err = redis.String(c.Do("SET", lock, value, "EX", expire, "NX"))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Unlock 解锁
func (s *RedisTemplate) Unlock(key, value string) (err error) {
	c := s.Pool.Get()
	defer func(c redis.Conn) {
		err := c.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: Unlock redis.Conn-close:err:%v------------", err)
		}
	}(c)
	//获取锁value
	setValue, err := redis.String(c.Do("GET", key))
	if err != nil {
		fmt.Printf("------------RedisTemplate: Unlock GET:err:%v------------", err)
		return
	}
	//判断锁是否属于该释放锁的线程
	if setValue != value {
		err = errors.New("非法用户，无法释放该锁")
		return
	}
	//属于该用户，直接删除该key
	_, err = c.Do("DEL", key)
	if err != nil {
		fmt.Printf("------------RedisTemplate: Unlock DEL:err:%v------------", err)
		return
	}
	return
}

// Incr 自增
func (s *RedisTemplate) Incr(key string) (result int, err error) {
	conn := s.Pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("------------RedisTemplate: Incr redis.Conn-close:err:%v------------", err)
		}
	}(conn)

	result, err = redis.Int(conn.Do("INCR", key))
	if err != nil {
		fmt.Printf("------------RedisTemplate: Incr INCR:err:%v------------", err)
		return result, err
	}
	return result, nil
}
