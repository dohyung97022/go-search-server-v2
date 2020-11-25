package main

import (
	"fmt"
	"testing"

	msqlf "github.com/dohyung97022/mysqlfunc"
)

func TestMain(t *testing.T) {
	err := msqlf.Init("dohyung97022", "9347314da!", "adiy-db.cxdzwqqcqoib.us-east-1.rds.amazonaws.com", 3306, "adiy")
	if err != nil {
		fmt.Printf("error : %v\n", err)
		logger.Println(err.Error())
		return
	}
	v, err := msqlf.GetDataOfWhere("search", []string{"last_update", "srch_id"},
		[]msqlf.Where{msqlf.Where{A: "query", IS: "=", B: "공포게임"}})
	if err != nil {
		fmt.Printf("error : %v\n", err)
		logger.Println(err.Error())
		return
	}
	fmt.Printf("v val = %v\n", v)
}

func Test2(t *testing.T) {
	mysql, err := newMysql()
	if err != nil {
		//too many connections error
		logger.Printf(err.Error())
		return
	}
	v, err := mysql.getIntStrStrMap.query(`SELECT * FROM adiy.search`)
	if err != nil {
		fmt.Printf("error : %v\n", err)
		logger.Println(err.Error())
		return
	}
	fmt.Printf("v val = %v\n", v)
}
