package database

import (
	"database/sql"
	"fmt"
	_"github.com/go-sql-driver/mysql"
	"log"
)

// 保存数据库连接
var dbConn *sql.DB

//数据库连接信息
const (
	SERVER = "118.190.175.14"
	USERNAME = "root"
	PASSWORD = "root"
	NETWORK = "tcp"
	PORT = 3306
	DATABASE = "douban"
	CHARSET = "utf8mb4"
)

// 连接数据库
func NewConnect() *sql.DB{
	if dbConn != nil {
		return dbConn
	}

	conn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=%s", USERNAME, PASSWORD, NETWORK, SERVER, PORT, DATABASE, CHARSET)
	DB, err := sql.Open("mysql", conn)
	if err != nil {
		log.Fatalln("数据库连接失败")
	}

	log.Println("数据库连接成功")

	dbConn = DB

	return dbConn
}

