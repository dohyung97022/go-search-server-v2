package main

import (
	"fmt"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"os"
)
//Mysql ----------------------------------------------------------
type Mysql struct {
	DB *sql.DB
	execute mysqlExecute
	getStr mysqlGetStr
	getStrAry mysqlGetStrAry
}
//mysql constructor
func newMysql()(mysql Mysql, err error){
	err = godotenv.Load("../mysql.env")
	if err != nil {
		return mysql, err
	}
	DB, err := sql.Open("mysql", os.Getenv("ID")+":"+os.Getenv("PS")+"@tcp("+os.Getenv("ENDPOINT")+":"+os.Getenv("PORT")+")/"+os.Getenv("SCHEMA")+"?multiStatements=true")
	if err != nil {
		return mysql, err
	}
	err = DB.Ping()
	if err != nil {
		return mysql, err
	}
	mysql.DB = DB
	mysql.execute.DB = DB
	mysql.getStr.DB = DB
	mysql.getStrAry.DB = DB
	return mysql, nil
}

//mysql.execute
type mysqlExecute struct {
	DB *sql.DB
}
//mysql.execute.query
func (mysql *mysqlExecute) query (queryStr string) (err error) {
	_, err = mysql.DB.Exec(queryStr)
	if err != nil {
		return err
	}
	return nil
}

type mysqlGetStr struct {
	DB *sql.DB
}
//mysql.getStr.paymentIDFromUID
func (mysql *mysqlGetStr) paymentIDFromUID (UIDStr string) (paymentIDStr string, err error){
	rows, err := mysql.DB.Query(`SELECT * FROM adiy.firebase_uid WHERE uid = "`+UIDStr+`"`)
	if err != nil {
		return "", err
	}
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	fmt.Printf("col = %v\n", columns)
	return "", nil
}

//mysql.getStrAry
type mysqlGetStrAry struct {
	DB *sql.DB
}
//mysql.getStrAry.query
func (mysql *mysqlGetStrAry) query (queryStr string)(resStrAry []string, err error){
	rows, err := mysql.DB.Query(queryStr)
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	return columns, nil
}