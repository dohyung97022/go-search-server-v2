package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

//Mysql ----------------------------------------------------------
type Mysql struct {
	DB              *sql.DB
	execute         mysqlExecute
	getStr          mysqlGetStr
	getIntStrStrMap mysqlGetIntStrStrMap
}

//mysql constructor
func newMysql() (mysql Mysql, err error) {
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
	mysql.getIntStrStrMap.DB = DB
	return mysql, nil
}

//mysql.execute
type mysqlExecute struct {
	DB *sql.DB
}

//mysql.execute.query
func (mysql *mysqlExecute) query(queryStr string) (err error) {
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
func (mysql *mysqlGetStr) paymentIDFromUID(UIDStr string) (paymentIDStr string, err error) {
	fmt.Printf("uid str := %v\n", UIDStr)
	rows, err := mysql.DB.Query(`SELECT * FROM adiy.firebase_uid WHERE uid = "` + UIDStr + `"`)
	defer rows.Close()
	if err != nil {
		return "", err
	}
	for rows.Next() {
		err = rows.Scan(&UIDStr, &paymentIDStr)
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("paymentIDString := %v\n", paymentIDStr)
	}
	return paymentIDStr, nil
}

//mysql.getStrStrMap
type mysqlGetIntStrStrMap struct {
	DB *sql.DB
}

//mysql.getIntStrStrMap.query
func (mysql *mysqlGetIntStrStrMap) query(queryStr string) (intStrStrMap map[int]map[string]string, err error) {
	intStrStrMap = make(map[int]map[string]string)
	rows, err := mysql.DB.Query(queryStr)
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
	}
	colNames, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}
	colLength := len(colNames)
	values := make([]interface{}, colLength)
	valuePtrs := make([]interface{}, colLength)
	rowCount := 0
	for rows.Next() {
		for i := 0; i < colLength; i++ {
			valuePtrs[i] = &values[i]

		}
		err = rows.Scan(valuePtrs...)
		if err != nil {
			log.Fatal(err)
		}
		strStrMap := make(map[string]string)
		for i, currentValue := range values {
			byteValue, done := currentValue.([]byte)
			if done {
				strStrMap[colNames[i]] = string(byteValue)
			}
		}
		intStrStrMap[rowCount] = strStrMap
		rowCount++
	}
	return intStrStrMap, nil
}
