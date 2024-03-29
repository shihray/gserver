package registry

import (
	"github.com/gomodule/redigo/redis"
	hash "github.com/mitchellh/hashstructure"
	MyRedisUtil "github.com/shihray/gserver/source/redis_util"
	log "github.com/z9905080/gloger"
	"strings"
	"sync"
)

const (
	Del      = "DEL"
	SRem     = "SREM"
	Keys     = "KEYS"
	SAdd     = "SADD"
	Set      = "SET"
	Get      = "GET"
	SMembers = "SMEMBERS"
)

type redisRegistry struct {
	*sync.RWMutex // Lock
	Address       string
	opts          Options
	connect       bool // connect enabled
	register      map[string]uint64
}

type RedisService struct {
	Name    string // service name
	Address string // service address
}

func newRedisRegistry(opts ...Option) Registry {
	cr := &redisRegistry{
		opts: Options{
			RedisHost:     "localhost:6379",
			RedisPassword: "",
			GroupID:       1,
		},
		RWMutex:  new(sync.RWMutex),
		register: make(map[string]uint64),
	}
	for _, o := range opts {
		o(&cr.opts)
	}
	return cr
}

func (c *redisRegistry) Deregister(s *Service) error {
	// delete the service
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return connErr
	}
	defer redisConn.Close()

	_, err := redisConn.Do(SRem, RegisterRedisKey.Title(c.Options().GroupID).Addr(s.Name), s.ID)
	if err != nil && err != redis.ErrNil {
		msg := "redis SREM Error, func: Deregister, 刪除陣列中元素錯誤 "
		log.Error(msg + err.Error())
		return err
	}
	if _, er := redisConn.Do(Del, ModuleInfoRedisKey.Title(c.Options().GroupID).Addr(s.ID)); er != nil {
		msg := "redis DEL Error, func: Deregister, 刪除Key錯誤 "
		log.Error(msg + err.Error())
		return err
	}

	// delete our hash of the service
	c.Lock()
	delete(c.register, s.ID)
	c.Unlock()

	return nil
}

func (c *redisRegistry) Register(s *Service, opts ...RegisterOption) error {
	// create hash of service; uint64
	h, err := hash.Hash(s, nil)
	if err != nil {
		return err
	}
	// get existing hash
	c.Lock()
	v, ok := c.register[s.Name]
	c.Unlock()

	// if it's already registered and matches then just pass the check
	if ok && v == h {
		return nil
	}
	// register the service
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return connErr
	}
	defer redisConn.Close()

	//redisName := fmt.Sprintf("%v@%v", s.Name, s.Address)
	if _, errOfRedis := redisConn.Do(SAdd, RegisterRedisKey.Title(c.Options().GroupID).Addr(s.Name), s.ID); errOfRedis != nil {
		msg := "redis SADD Error, func: Register, 加入列表失敗 "
		log.Error(msg + errOfRedis.Error())
		return nil
	}
	if _, setKeyErr := redisConn.Do(Set, ModuleInfoRedisKey.Title(c.Options().GroupID).Addr(s.ID), s.Address); setKeyErr != nil {
		msg := "redis SET Error, func: Register, 加入列表失敗 "
		log.Error(msg + setKeyErr.Error())
		return nil
	}
	// save our hash of the service
	c.Lock()
	c.register[s.ID] = h
	c.Unlock()

	log.DebugF("[%v]Redis Register Success", s.ID)

	return nil
}

func (c *redisRegistry) GetService(name string) ([]*Service, error) {
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return nil, connErr
	}
	defer redisConn.Close()

	hList := make(map[string]string, 0)
	nameSplit := strings.Split(name, "@")
	if len(nameSplit) == 1 {
		moduleList, err := redis.Strings(redisConn.Do(SMembers, RegisterRedisKey.Title(c.Options().GroupID).Addr(name)))
		if err != nil {
			msg := "redis SMEMBERS Error, func: GetService, 取得Redis資料Key錯誤 "
			log.Error(msg + err.Error())
			return nil, err
		}
		for _, s := range moduleList {
			addr, err := redis.String(redisConn.Do(Get, ModuleInfoRedisKey.Title(c.Options().GroupID).Addr(s)))
			if err == redis.ErrNil {
				_, err := redisConn.Do(SRem, RegisterRedisKey.Title(c.Options().GroupID).Addr(nameSplit[0]), name)
				if err != nil && err != redis.ErrNil {
					msg := "redis SREM Error, func: GetService, 刪除陣列中元素錯誤 "
					log.Error(msg + err.Error())
					return nil, err
				}
				continue
			}
			if err != nil {
				msg := "redis GET Error, func: GetService, 取得Redis資料Key錯誤 "
				log.Error(msg + err.Error())
				return nil, err
			}
			hList[s] = addr
		}
	} else {
		addr, err := redis.String(redisConn.Do(Get, ModuleInfoRedisKey.Title(c.Options().GroupID).Addr(name)))
		if err == redis.ErrNil {
			_, err := redisConn.Do(SRem, RegisterRedisKey.Title(c.Options().GroupID).Addr(nameSplit[0]), name)
			if err != nil && err != redis.ErrNil {
				msg := "redis SREM Error, func: GetService, 刪除陣列中元素錯誤 "
				log.Error(msg + err.Error())
				return nil, err
			}
			return nil, err
		}
		if err != nil {
			msg := "redis GET Error, func: GetService, 取得Redis資料Key錯誤 "
			log.Error(msg + err.Error())
			return nil, err
		}
		hList[name] = addr
	}

	log.DebugF("Redis GetService:%v", hList)

	var services []*Service
	for redisName, address := range hList {
		svc := &Service{
			Name:    name,
			ID:      redisName,
			Address: address,
		}
		services = append(services, svc)
	}

	return services, nil
}

func (c *redisRegistry) ListServices() ([]*Service, error) {
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return nil, connErr
	}
	defer redisConn.Close()

	keys, findKeysErr := redis.Strings(redisConn.Do(Keys, RegisterRedisKey.Title(c.Options().GroupID).Addr("*")))
	if findKeysErr != nil {
		return nil, findKeysErr
	}

	var services []*Service
	for _, key := range keys {
		name := strings.Split(key, ":")
		c.Clean(name[len(name)-1])
		serviceList, getListErr := redis.Strings(redisConn.Do(SMembers, key))
		if getListErr != nil {
			msg := "redis KEYS Error, func: GetService, 取得Redis資料Key錯誤 "
			log.Error(msg + getListErr.Error())
			return nil, getListErr
		}

		for _, id := range serviceList {
			addr, getErr := redis.String(redisConn.Do(Get, ModuleInfoRedisKey.Title(c.Options().GroupID).Addr(id)))
			if getErr != nil {
				msg := "redis GET Error, func: GetService, 取得Redis資料錯誤 "
				log.Error(msg + getErr.Error())
				return nil, getErr
			}
			services = append(services, &Service{
				ID:      id,
				Name:    name[len(name)-1],
				Address: addr,
			})
		}
	}
	return services, nil
}

func (c *redisRegistry) String() string {
	return "redisRegistry"
}

func (c *redisRegistry) Options() Options {
	return c.opts
}

func (c *redisRegistry) ConnectRedis() (redis.Conn, error) {
	redisPool, getPoolErr := MyRedisUtil.NewConnect(c.Options().RedisHost, c.Options().RedisPassword)
	if getPoolErr != nil {
		msg := "redis 連線錯誤,func: Register, 取得連線池失敗 "
		log.Error(msg + getPoolErr.Error())
		return nil, getPoolErr
	}
	return redisPool.Get(), getPoolErr
}

func (c *redisRegistry) Clean(typeName string) error {
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return connErr
	}
	defer redisConn.Close()
	// 檢查清單中ID
	isExistMap := make(map[string]bool, 0)
	moduleKeys, moduleKeysErr := redis.Strings(redisConn.Do(SMembers, RegisterRedisKey.Title(c.Options().GroupID).Addr(typeName)))
	if moduleKeysErr != nil {
		return moduleKeysErr
	}
	for _, key := range moduleKeys {
		isExistMap[key] = true
	}
	// 找尋所有module Info
	keys, infoKeysErr := redis.Strings(redisConn.Do(Keys, ModuleInfoRedisKey.Title(c.Options().GroupID).Addr(typeName+"*")))
	if infoKeysErr != nil {
		return infoKeysErr
	}
	// 比對list & Infos 資料，並將找不到的移除
	for _, key := range keys {
		// gserver:module:info:module_type@hashID
		splitKey := strings.Split(key, ":")
		// module_type@hashID
		name := splitKey[len(splitKey)-1]
		if _, ok := isExistMap[name]; !ok {
			// 移除list中未對應 address資料
			if _, sremErr := redisConn.Do(SRem, RegisterRedisKey.Title(c.Options().GroupID).Addr(typeName), name); sremErr != nil {
				return sremErr
			}
			// 刪除儲存address欄位
			if _, delErr := redisConn.Do(Del, key); delErr != nil {
				return delErr
			}
		}
	}

	return nil
}

func (c *redisRegistry) Check(id string) bool {

	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return false
	}
	defer redisConn.Close()

	idSplit := strings.Split(id, "@")
	if len(idSplit) != 2 {
		log.Error("資料長度錯誤:", id)
		return false
	}

	_, err := redisConn.Do("SISMEMBER", RegisterRedisKey.Title(c.Options().GroupID).Addr(idSplit[0]), id)
	if err != nil {
		msg := "redis SISMEMBER Error, func: Check,檢查陣列中元素錯誤 "
		log.Error(msg + err.Error())
		return false
	}

	if _, getModuleErr := redisConn.Do(Get, ModuleInfoRedisKey.Title(c.Options().GroupID).Addr(id)); getModuleErr != nil {
		log.ErrorF("redis Get Err:%v", getModuleErr.Error())
		return false
	}

	return true
}
