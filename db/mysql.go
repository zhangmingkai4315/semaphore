package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql" // imports mysql driver
	"github.com/zhangmingkai4315/semaphore/util"
	"gopkg.in/gorp.v1"
)

var Mysql *gorp.DbMap

// Mysql database
func Connect() error {
	db, err := connect()
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		if err := createDb(); err != nil {
			return err
		}

		db, err = connect()
		if err != nil {
			return err
		}

		if err := db.Ping(); err != nil {
			return err
		}
	}

	Mysql = &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}}
	return nil
}

func createDb() error {
	cfg := util.Config.MySQL
	url := cfg.Username + ":" + cfg.Password + "@tcp(" + cfg.Hostname + ")/?parseTime=true&interpolateParams=true"

	db, err := sql.Open("mysql", url)
	if err != nil {
		return err
	}

	if _, err := db.Exec("create database if not exists " + cfg.DbName); err != nil {
		return err
	}

	return nil
}

func connect() (*sql.DB, error) {
	cfg := util.Config.MySQL
	url := cfg.Username + ":" + cfg.Password + "@tcp(" + cfg.Hostname + ")/" + cfg.DbName + "?parseTime=true&interpolateParams=true"

	return sql.Open("mysql", url)
}
