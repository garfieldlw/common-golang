package sqlite3

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
	"sync"
	"time"
)

type DbInfo struct {
	DbConfig *ConfigItem
	Conn     *gorm.DB
}

var instance *DbInfo
var lock = &sync.Mutex{}

func LoadDb() *DbInfo {
	if instance != nil {
		return instance
	}

	lock.Lock()
	defer lock.Unlock()

	if instance == nil {
		dbConf := getDbConfig()
		if dbConf == nil {
			return nil
		}

		dnInfo := new(DbInfo)
		dnInfo.DbConfig = dbConf
		dnInfo.InitConnect()

		instance = dnInfo
	}

	return instance
}

func (info *DbInfo) InitConnect() {
	dbConf := info.DbConfig

	dsn := fmt.Sprintf("%s:%s?cache=shared&mode=rwc", dbConf.Type, dbConf.Path)
	if len(dbConf.User) > 0 {
		dsn = fmt.Sprintf("%s&_auth&_auth_user=%s&_auth_pass=%s", dsn, dbConf.User, dbConf.Password)
	}

	if db, e := gorm.Open(sqlite.Open(dsn), &gorm.Config{}); e != nil {
		info.Conn = nil
	} else {
		if dbConf.Idle == 0 {
			dbConf.Idle = 5
		}
		if dbConf.Open == 0 {
			dbConf.Open = 20
		}
		_ = db.Use(dbresolver.Register(dbresolver.Config{}).SetMaxIdleConns(int(dbConf.Idle)).SetMaxOpenConns(int(dbConf.Open)).SetConnMaxLifetime(time.Second * 3300))
		db = db.Session(&gorm.Session{})
		info.Conn = db
	}
}

func (info *DbInfo) CheckAndReturnConn() *gorm.DB {
	if info.Conn == nil {
		lock.Lock()
		defer lock.Unlock()
		if info.Conn == nil || info.Conn.Error != nil {
			info.InitConnect()
		}
	}
	if info.Conn == nil {
		return nil
	}

	return info.Conn
}

func GetDb() *gorm.DB {
	ins := LoadDb()
	if ins == nil {
		return nil
	}
	return ins.CheckAndReturnConn()
}

type ConfigItem struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	User     string `json:"user"`
	Password string `json:"password"`
	Open     int64  `json:"open"`
	Idle     int64  `json:"idle"`
}

func getDbConfig() *ConfigItem {
	return &ConfigItem{
		Type:     "file",
		Path:     "db.sqlite3",
		User:     "",
		Password: "",
		Open:     0,
		Idle:     0,
	}
}
