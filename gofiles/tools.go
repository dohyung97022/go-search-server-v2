package main

import(
	"strconv"
)

//Tools ----------------------------------------------------------
type Tools struct{
	getStrAry toolsGetStrAry
	getStr toolsGetStr
	getInt toolsGetInt
}
var tools Tools
//getInt
type toolsGetInt struct{}
//getInt.fromStr
func (*toolsGetInt) fromStr(fromStr string)(resInt int, err error){
	resInt, err = strconv.Atoi(fromStr)
	if err!= nil{
		logger.Printf(err.Error())
		return 0, err
	}
	return
}

//getStr
type toolsGetStr struct{}
//getStr.fromInt
func (*toolsGetStr) fromInt(fromInt int)(resStr string){
	return strconv.Itoa(fromInt)
}

//getStrAry
type toolsGetStrAry struct{}
//getStrAry.noDuplicate
func (*toolsGetStrAry) noDuplicate(strAry []string) (res []string){
	noDuplicate := make(map[string]bool)
	for _, str := range strAry {
		if _, wasInserted := noDuplicate[str]; !wasInserted {
		noDuplicate[str]=true
		res = append(res, str)
		}
	}
	return res
}
//getStrAry.combind
func (*toolsGetStrAry) combind(strAry1 []string, strAry2 []string)(res []string){
	for _,str := range strAry1{
		res = append(res,str)
	}
	for _,str := range strAry2{
		res = append(res,str)
	}
	return res
} 