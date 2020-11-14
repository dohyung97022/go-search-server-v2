package main

import(
	"net/http"
	"io/ioutil"
	"strings"
	"net/url"
	"errors"
	"io"
	"encoding/json"
)
//Curl ----------------------------------------------------------
type Curl struct {
	getStr curlGetStr 
	getInterface curlGetInterface
	setRequest curlSetRequest
}
var curl Curl
//getRequest
func (*Curl) getRequest(urlStr string, AllCapsMethod string, body io.Reader) (request *http.Request, err error){
	request, err = http.NewRequest(AllCapsMethod, urlStr, body)
	if err != nil {
		logger.Println(err.Error())
		return nil, err
	}
	return request, err
}
//getStr
type curlGetStr struct{}
//getStr.results
func (*curlGetStr) results(urlStr string) (resStr string, err error){
	if urlStr == ""{
		errStr := `error : getStr.results got urlStr of ""`
		logger.Printf(errStr)
		return "" , errors.New(errStr)
	}
	response, err := http.Get(urlStr)
	if err != nil {
		logger.Println(err.Error())
		return "", err
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Println(err.Error())
		return "", err
	}
	defer response.Body.Close()
	return string(body), nil 
}
//getStr.requestResults
func (*curlGetStr) requestResults(request *http.Request) (resStr string, err error){
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		logger.Println(err.Error())
		return
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Println(err.Error())
		return "", err
	}
	defer response.Body.Close()
	return string(body), nil 
}
//getStr.formatedURL
func (*curlGetStr) formatedURL(urlStr string)(resStr string){
	return strings.ReplaceAll(urlStr,"\\","")
}
//getStr.decoded
func (*curlGetStr) decoded(encodedStr string)(resStr string, err error){
	resStr, err = url.QueryUnescape(encodedStr)
	if err != nil {
		logger.Println(err.Error())
		return "", err
	}
	return resStr, err
}

//getInterface
type curlGetInterface struct{}
//getInterface.requestResults
func (*curlGetInterface) requestResults(request *http.Request, resInterf interface{}) (err error){
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		logger.Println(err.Error())
		return
	}
	// body, err := ioutil.ReadAll(response.Body)
	// if err != nil {
	// 	logger.Println(err.Error())
	// 	return err
	// }
	defer response.Body.Close()
	return  json.NewDecoder(response.Body).Decode(resInterf)
}



//setRequest
type curlSetRequest struct{}
//setRequest.header
func (*curlSetRequest) header(request *http.Request, keyStr string, valStr string){
	request.Header.Set(keyStr, valStr)
}
//setRequest.userNamePassword
func (*curlSetRequest) userNamePassword(request *http.Request, userNameStr string, passwordStr string){
	request.SetBasicAuth(userNameStr, passwordStr)
}
