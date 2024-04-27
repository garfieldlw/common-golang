package mysql

import (
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
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

func LoadPgDb() *DbInfo {
	if instance != nil {
		return instance
	}

	lock.Lock()
	defer lock.Unlock()

	if instance == nil {
		dbConf := getMysqlConfig()
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
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai", dbConf.Host, dbConf.User, dbConf.Password, dbConf.Database, dbConf.Port)

	if db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{}); e != nil {
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
	pgIns := LoadPgDb()
	if pgIns == nil {
		return nil
	}
	return pgIns.CheckAndReturnConn()
}

func Ping() error {
	conn := GetDb()
	if conn == nil {
		return errors.New("connect mysql fail")
	}

	if db, err := conn.DB(); err == nil {
		err = db.Ping()
		if err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

type ConfigItem struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Open     int64  `json:"open"`
	Idle     int64  `json:"idle"`
}

func getMysqlConfig() *ConfigItem {
	return &ConfigItem{
		Host:     "127.0.0.1",
		Port:     "3306",
		User:     "root",
		Password: "password",
		Database: "123456",
		Open:     0,
		Idle:     0,
	}
}
