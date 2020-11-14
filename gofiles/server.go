package main

import(
	"net/http"
	"errors"
	"fmt"
)

//Server ----------------------------------------------------------
type Server struct {
	w *http.ResponseWriter
	r *http.Request
	getStr serverGetStr
	setStr serverSetStr
	execStr serverExecStr
}
//server constructor
func newServer(w *http.ResponseWriter, r *http.Request) (server Server) {
	server.w = w
	server.r = r
	server.getStr.w = w
	server.getStr.r = r
	server.setStr.w = w
	server.setStr.r = r
	server.execStr.w = w
	server.execStr.r = r
	return server
}
//server.execStr
type serverExecStr struct {
	w *http.ResponseWriter
	r *http.Request
}
func (server serverExecStr) respond (respondStr string) {
	fmt.Fprintf(*server.w, "%s", respondStr)
}

//server.getStr
type serverGetStr struct {
	w *http.ResponseWriter
	r *http.Request
}
//server.getStr.query
func (server serverGetStr) query (queryStr string) (resStr string, err error){
	params, ok := server.r.URL.Query()[queryStr]
	if !ok || len(params) == 0 {
		err = errors.New("error : no such parameter of "+ queryStr +" was found.")
		logger.Printf(err.Error())
		return "", err
	}
	return params[0], nil
}
//server.getStr.queryOrDefault
func (server serverGetStr) queryOrDefault(queryStr string, defaultStr string) string{
	params, ok := server.r.URL.Query()[queryStr]
	if !ok || len(params) == 0 {
		return defaultStr
	}
	return params[0]
}
//server.getStr.header
func (server serverGetStr) header(headerNameStr string) (resStr string, err error){
	resStr = server.r.Header.Get(headerNameStr)
	if resStr == ""{
		err = errors.New("error : no such header value of "+ headerNameStr +" was found.")
		logger.Printf(err.Error())
		return resStr, err
	}
	return resStr, nil
}

//server.setStr
type serverSetStr struct {
	w *http.ResponseWriter
	r *http.Request
}
//server.setStr.header
func (server serverSetStr) header(headerNameStr string, headerValueStr string){
	(*server.w).Header().Set(headerNameStr, headerValueStr)
}