package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	msqlf "github.com/dohyung97022/mysqlfunc"
)

// --------------------------------- struct types --------------------------------------
type callScraperStruct struct {
	//ioutil or goquery
	Type string
	//urls to scrape
	Urls []string
}

// --------------------------------- mutex var --------------------------------------
var (
	lambdaCountUID = 100
	mutex          sync.Mutex
)

// --------------------------------- logger var --------------------------------------
var (
	loggerFile, _ = os.OpenFile("err.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logger        = log.New(loggerFile, "Log", log.LstdFlags|log.Lshortfile)
)

// --------------------------------- server functions --------------------------------------
//http://ec2-54-161-234-228.compute-1.amazonaws.com:3000/search?search=
// ec2-54-161-234-228.compute-1.amazonaws.com
// http://localhost:3000/search?search=
func main() {
	err := msqlf.Init("dohyung97022", "9347314da!", "adiy-db.cxdzwqqcqoib.us-east-1.rds.amazonaws.com", 3306, "adiy")
	if err != nil {
		fmt.Printf("error : %v\n", err)
		return
	}
	fmt.Println("server is up and running")
	http.HandleFunc("/search", handler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}
func handler(w http.ResponseWriter, r *http.Request) {
	// ----------------- execution time -----------------
	fmt.Println("request on 3000 (search)")
	start := time.Now()
	defer func() {
		fmt.Printf("Binomial took %v\n", time.Since(start))
	}()

	// ----------------- parameters -----------------
	params, ok := r.URL.Query()["search"]
	if !ok || params[0] == "" {
		log.Println("Url Param 'search' is missing")
		fmt.Fprintf(w, "You are missing the 'search' param!\n")
		return
	}
	search := params[0]
	v, err := msqlf.GetDataOfWhere("search", []string{"last_update", "srch_id"},
		[]msqlf.Where{msqlf.Where{A: "query", IS: "=", B: search}})
	if err != nil {
		fmt.Printf("error : %v\n", err)
		return
	}
	// avMax := queryIntDefaultZero("avmax", r)
	// avMin := queryIntDefaultZero("avmin", r)
	// sbMax := queryIntDefaultZero("sbmax", r)
	// sbMin := queryIntDefaultZero("sbmin", r)
	// tvMax := queryIntDefaultZero("tvmin", r)
	// tvMin := queryIntDefaultZero("tvmax", r)

	// ----------------- need scraping? -----------------
	needRef := false
	// srchID := -1
	// search table has no data of query
	if len(v) == 0 {
		needRef = true
		// search table has data of query
	} else {
		t, err := time.Parse("2006-01-02 15:04:05", v[0]["last_update"].(string))
		if err != nil {
			fmt.Printf("error : %v\n", err)
			return
		}
		// outdated, but has search data
		if t.Before(time.Now().AddDate(0, 0, -2)) {
			needRef = true
			// srchID = v[0]["srch_id"].(int)
		}
	}

	intInfo := make(map[int]info)
	// channelBools := make(map[string]bool)
	if needRef {
		_, intInfo, err = scrape(search)
		if err != nil {
			fmt.Printf("error : %v\n", err)
			return
		}
		resultInterface := make(map[int]map[string]interface{})
		for intt, info := range intInfo {
			resultInterface[intt] = make(map[string]interface{})
			resultInterface[intt]["ChanUrl"] = info.ChanURL
			resultInterface[intt]["Channel"] = info.Channel
			resultInterface[intt]["Title"] = info.Title
			resultInterface[intt]["ChanImg"] = info.ChanImg
			resultInterface[intt]["About"] = info.About
			resultInterface[intt]["Subs"] = info.Subs
			resultInterface[intt]["Views"] = info.Views
			resultInterface[intt]["AvrViews"] = info.AvrViews
			resultInterface[intt]["UploadTime"] = info.UploadTime
			resultInterface[intt]["Links"] = info.Links
		}
		fmt.Printf("len resultInterface : %v\n", len(resultInterface))
	} else {
		// get from search.
	}

	//데이터가 존재했었다. 이전의 업데이트가 부족한 체널들 scrape
	// if srchID != -1 {
	// 	go scrapeOutdated(search, channelBools, intInfo)
	// }

	// checking if search data is outdated

}
func queryIntDefaultZero(query string, r *http.Request) int {
	params, ok := r.URL.Query()[query]
	if !ok || len(params) == 0 {
		return 0
	}
	value, err := strconv.Atoi(params[0])
	if err != nil {
		return 0
	}
	return value
}

// --------------------------------- scrape functions --------------------------------------
func scrape(search string) (stringBoolChannels map[string]bool, intInfo map[int]info, err error) {
	search, _ = url.PathUnescape(search)
	search = strings.ReplaceAll(search, " ", "+")
	urlsArray := []string{
		"https://www.youtube.com/results?search_query=" + search,
		"https://www.youtube.com/results?page=2&search_query=" + search,
		"https://www.youtube.com/results?page=3&search_query=" + search,
		"https://www.youtube.com/results?page=4&search_query=" + search,
		"https://www.youtube.com/results?page=5&search_query=" + search,
		"https://www.youtube.com/results?page=6&search_query=" + search,
		"https://www.youtube.com/results?page=7&search_query=" + search,
		"https://www.youtube.com/results?page=8&search_query=" + search,
		"https://www.youtube.com/results?page=9&search_query=" + search,
		"https://www.youtube.com/results?page=10&search_query=" + search,
		"https://www.youtube.com/results?page=11&search_query=" + search,
		"https://www.youtube.com/results?page=12&search_query=" + search,
		"https://www.youtube.com/results?page=13&search_query=" + search,
		"https://www.youtube.com/results?page=14&search_query=" + search,
		"https://www.youtube.com/results?page=15&search_query=" + search,
		"https://www.youtube.com/results?page=16&search_query=" + search,
		"https://www.youtube.com/results?page=17&search_query=" + search,
		"https://www.youtube.com/results?page=18&search_query=" + search,
		"https://www.youtube.com/results?page=19&search_query=" + search,
		"https://www.youtube.com/results?page=20&search_query=" + search}

	URLStringScript := callScraperHandler(urlsArray, "ioutil")
	stringBoolChannels = findChannelsHandler(URLStringScript)

	aboutUrlsArray := []string{}
	for channel := range stringBoolChannels {
		aboutUrlsArray = append(aboutUrlsArray, "https://www.youtube.com"+channel+"/about")
	}
	videosUrlsArray := []string{}
	for channel := range stringBoolChannels {
		videosUrlsArray = append(videosUrlsArray, "https://www.youtube.com"+channel+"/videos")
	}
	chAbout := make(chan map[string]string)
	chVideos := make(chan map[string]string)
	go func() { chAbout <- callScraperHandler(aboutUrlsArray, "goquery") }()
	go func() { chVideos <- callScraperHandler(videosUrlsArray, "goquery") }()

	URLScriptAbout := <-chAbout
	URLScriptVideos := <-chVideos

	chanInfo := findInfoHandler(URLScriptAbout)
	chanVideosInfo := findVideosInfoHandler(URLScriptVideos)

	i := 0
	intInfo = make(map[int]info)
	for url, info := range chanInfo {
		info.AvrViews = chanVideosInfo[url].AvrViews
		info.UploadTime = chanVideosInfo[url].UploadTime
		intInfo[i] = info
		i++
	}

	return stringBoolChannels, intInfo, nil
}

// func scrapeOutdated(search string, channelBools map[string]bool, intInfo map[int]info) {
// 	start := time.Now()
// 	// 1. 이미 등록되었는데 검색되지 않은 체널 확인
// 	needRefChan, err := getFbSearchRelChansThatNeedsRefresh(search, time.Now().AddDate(0, 0, -1).UTC())
// 	if err != nil {
// 		fmt.Printf("error := %v\n", err.Error())
// 		logger.Printf("error := %v\n", err.Error())
// 	}
// 	timeTrack(start, "getFbSearchRelChansThatNeedsRefresh")
// 	start = time.Now()

// 	var crawlResChans []string
// 	for str := range channelBools {
// 		crawlResChans = append(crawlResChans, str)
// 	}
// 	needRefChan = subtractStrArray(needRefChan, crawlResChans)

// 	// 2. 체널 정보 받기
// 	//이제는 스크레이프를 하고 받아야 한다.!
// 	aboutUrlsArray := []string{}
// 	for _, channel := range needRefChan {
// 		aboutUrlsArray = append(aboutUrlsArray, "https://www.youtube.com"+channel+"/about")
// 	}
// 	videosUrlsArray := []string{}
// 	for _, channel := range needRefChan {
// 		videosUrlsArray = append(videosUrlsArray, "https://www.youtube.com"+channel+"/videos")
// 	}
// 	chAbout := make(chan map[string]string)
// 	chVideos := make(chan map[string]string)
// 	go func() { chAbout <- callScraperHandler(aboutUrlsArray, "goquery") }()
// 	go func() { chVideos <- callScraperHandler(videosUrlsArray, "goquery") }()

// 	URLScriptAbout := <-chAbout
// 	URLScriptVideos := <-chVideos

// 	chanInfo := findInfoHandler(URLScriptAbout)
// 	chanVideosInfo := findVideosInfoHandler(URLScriptVideos)

// 	i := 0
// 	for url, info := range chanInfo {
// 		info.AvrViews = chanVideosInfo[url].AvrViews
// 		info.UploadTime = chanVideosInfo[url].UploadTime
// 		intInfo[len(intInfo)+i] = info
// 		i++
// 	}
// 	errAry := saveFbChanData(search, intInfo)
// 	timeTrack(start, "saveFbChanData")
// 	if len(errAry) != 0 {
// 		for _, err := range errAry {
// 			fmt.Printf("error := %v\n", err.Error())
// 			logger.Printf("error := %v\n", err.Error())
// 		}
// 	}
// }
func findChannelsHandler(urlScript map[string]string) (foundUrls map[string]bool) {
	foundUrls = make(map[string]bool)
	chUrls := make(chan []string)
	chFinished := make(chan bool)

	for _, s := range urlScript {
		go findChannels(s, chUrls, chFinished)
	}
	for c := 0; c < len(urlScript); {
		select {
		case url := <-chUrls:
			for i := range url {
				foundUrls[url[i]] = true
			}
		case <-chFinished:
			c++
		}
	}
	return foundUrls
}
func findChannels(s string, ch chan []string, chFinished chan bool) {
	var channels []string
	// capcha type 1
	if between(string(s), "<script>", "</script>") ==
		"var submitCallback = function(response) {document.getElementById('captcha-form').submit();};" {
		logger.Printf("Capcha has been detected. (crawlVideo) type 1 \n")
		fmt.Printf("error :%v\n", "Capcha has been detected. (crawlVideo) type 1")
	}
	// capcha type 2
	if strings.Contains(between(s, "<script src", "script>"), "https://www.google.com/recaptcha/api.js") == true {
		logger.Printf("Capcha has been detected. (crawlVideo) type 2\n")
		fmt.Printf("error :%v\n", "Capcha has been detected. (crawlVideo) type 2")
	}
	// capcha type 3
	if strings.Contains(between(s, "<script  src", "script>"), "https://www.google.com/recaptcha/api.js") == true {
		logger.Printf("Capcha has been detected. (crawlVideo) type 3\n")
		fmt.Printf("error :%v\n", "Capcha has been detected. (crawlVideo) type 3")
	}
	z := strings.Split(s, "\"commandMetadata\"")
	for val := range z {
		linkPre := between(z[val], "webCommandMetadata", "}")
		linkPre = between(linkPre, "{\"url\":\"", "\"")
		if strings.Index(linkPre, "/channel/") == 0 {
			if !contains(channels, linkPre) {
				channels = append(channels, linkPre)
			}
		}
		if strings.Index(linkPre, "/user/") == 0 {
			if !contains(channels, linkPre) {
				channels = append(channels, linkPre)
			}
		}
	}
	defer func() {
		ch <- channels
		chFinished <- true
	}()
}

type videosInfo struct {
	Channel    string
	UploadTime string
	AvrViews   int
}

func findVideosInfoHandler(urlScript map[string]string) (finalVideosInfo map[string]videosInfo) {
	finalVideosInfo = make(map[string]videosInfo)
	chVideosInfo := make(chan videosInfo)
	chFinished := make(chan bool)

	for url, s := range urlScript {
		go findVideosInfo(url, s, chVideosInfo, chFinished)
	}
	for c := 0; c < len(urlScript); {
		select {
		case videosInfo := <-chVideosInfo:
			finalVideosInfo[videosInfo.Channel] = videosInfo
		case <-chFinished:
			c++
		}
	}
	return finalVideosInfo
}
func findVideosInfo(url string, s string, chVideosInfo chan videosInfo, chFinished chan bool) {
	storeVideosInfo := videosInfo{
		Channel:    between(url, "https://www.youtube.com", "/videos"),
		UploadTime: "",
		AvrViews:   0,
	}
	viewsArray := strings.Split(s, "shortViewCountText\":{\"simpleText\":\"")
	// println(len(viewsArray))
	for i := range viewsArray {
		viewsArray[i] = before(viewsArray[i], "\"")
	}

	datesArray := strings.Split(s, "publishedTimeText\":{\"simpleText\":\"")
	// println(len(datesArray))
	for i := range datesArray {
		datesArray[i] = before(datesArray[i], "\"")
	}

	// storeInfo.AvrViews = viewsArray[1]
	if len(datesArray) > 1 {
		storeVideosInfo.UploadTime = datesArray[1]
	}
	p := 0
	var sum int
	var addcnt int
	if len(viewsArray) > len(datesArray) {
		p = len(datesArray)
	} else {
		p = len(viewsArray)
	}
	reg, _ := regexp.Compile("[^0-9.]+")
	for c := 0; c < p; c++ {
		if isWithinYear(datesArray[c]) == true {
			m := checkViewsMultiplyer(viewsArray[c])
			viewsArray[c] = reg.ReplaceAllString(viewsArray[c], "")
			viewFloat, err := strconv.ParseFloat(viewsArray[c], 64)
			viewInt := int(viewFloat * m)
			if err != nil {
			} else {
				addcnt++
				sum += viewInt
			}
		}
	}
	if addcnt == 0 {
		addcnt = 1
	}
	storeVideosInfo.AvrViews = sum / addcnt
	defer func() {
		chVideosInfo <- storeVideosInfo
		chFinished <- true
	}()
}

type info struct {
	ChanURL    string
	Channel    string
	Title      string
	ChanImg    string
	About      string
	Subs       int
	Views      int
	AvrViews   int
	UploadTime string
	Links      map[string]string
	Script     string
}

func findInfoHandler(urlScript map[string]string) (finalStringInfo map[string]info) {
	finalStringInfo = make(map[string]info)
	chanInfo := make(chan info)
	chFinished := make(chan bool)

	for url, s := range urlScript {
		go findInfo(url, s, chanInfo, chFinished)
	}
	for c := 0; c < len(urlScript); {
		select {
		case info := <-chanInfo:
			finalStringInfo[info.Channel] = info
		case <-chFinished:
			c++
		}
	}
	return finalStringInfo
}
func findInfo(chanURL string, s string, chanInfo chan info, chFinished chan bool) {
	storeInfo := info{
		ChanURL:    chanURL,
		Channel:    between(chanURL, "https://www.youtube.com", "/about"),
		Title:      "",
		ChanImg:    "",
		About:      "",
		Subs:       0,
		Views:      0,
		AvrViews:   0,
		UploadTime: "",
		Links: map[string]string{
			"FacebookGroup": "",
			"FacebookPage":  "",
			"Facebook":      "",
			"Twitch":        "",
			"Instagram":     "",
			"Twitter":       "",
		},
		Script: "",
	}
	// capcha type 1
	if between(string(s), "<script>", "</script>") ==
		"var submitCallback = function(response) {document.getElementById('captcha-form').submit();};" {
		logger.Printf("Capcha has been detected. (crawlVideo) type 1 \n")
		fmt.Printf("error :%v\n", "Capcha has been detected. (crawlVideo) type 1")
	}
	// capcha type 2
	if strings.Contains(between(s, "<script src", "script>"), "https://www.google.com/recaptcha/api.js") == true {
		logger.Printf("Capcha has been detected. (crawlVideo) type 2\n")
		fmt.Printf("error :%v\n", "Capcha has been detected. (crawlVideo) type 2")
	}
	// capcha type 3
	if strings.Contains(between(s, "<script  src", "script>"), "https://www.google.com/recaptcha/api.js") == true {
		logger.Printf("Capcha has been detected. (crawlVideo) type 3\n")
		fmt.Printf("error :%v\n", "Capcha has been detected. (crawlVideo) type 3")
	}
	defer func() {
		chanInfo <- storeInfo
		chFinished <- true
	}()
	// check autogenerated by youtube
	if strings.Contains(s, "Auto-generated by YouTube") {
		logger.Printf("Auto-generation has been detected\n")
		fmt.Printf("Auto-generation has been detected\n")
	}
	if strings.Contains(s, "autoGenerated") {
		logger.Printf("Auto-generation has been detected\n")
		fmt.Printf("Auto-generation has been detected\n")
	}

	// title
	title := between(s, "channelMetadataRenderer\":{\"title\":\"", "\"")
	storeInfo.Title += title
	//channel img
	img := between(s, "\"avatar\":{\"thumbnails\":[{\"url\":\"", "\"")
	storeInfo.ChanImg += img
	// total views
	views := between(s, "viewCountText", ",\"")
	storeInfo.Views += removeButNumber(views)
	// abouts
	abouts := after(s, "\"channelMetadataRenderer\":{\"title\":\"")
	abouts = between(abouts, "description\":\"", "\",\"")
	storeInfo.About += abouts
	//subs
	subs := after(s, "subscriberCountText")
	subs = before(subs, "\"}")
	subs = after(subs, "\":\"")
	subs = strings.Replace(subs, "subscribers", "", 1)
	storeInfo.Subs = subscriberStringToInt(subs)
	//links
	linksPre := between(s, "primaryLinks\":", "channelMetadataRenderer")
	linksArray := strings.Split(linksPre, "thumbnails")
	for val := range linksArray {
		link := after(linksArray[val], "urlEndpoint")
		link = between(link, "q=", "\"")
		if strings.Contains(link, "\\u0026") {
			link = before(link, "\\u0026")
		}
		decodedValue, _ := url.PathUnescape(link)
		if decodedValue != "" {
			//links url title
			title := between(linksArray[val], "title\":", "}}")
			title = between(title, ":\"", "\"")
			urlTitle, sucss := checkForSocial(decodedValue)
			if sucss {
				storeInfo.Links[urlTitle] = decodedValue
			}
		}
	}
	return
}

func callScraperHandler(urlArray []string, scrapeType string) (finalURLScripts map[string]string) {
	start := time.Now()
	finalURLScripts = make(map[string]string)
	chanURLScripts := make(chan map[string]string)
	chFinished := make(chan bool)

	devider := 3
	quotient, remainder := len(urlArray)/devider, len(urlArray)%devider
	for i := 0; i < quotient; i++ {
		go callScraper(urlArray[i*devider:((i+1)*devider)], scrapeType, chanURLScripts, chFinished)
	}
	if remainder != 0 {
		go callScraper(urlArray[quotient*devider:quotient*devider+remainder], scrapeType, chanURLScripts, chFinished)
		quotient++
	}
	for i := 0; i < quotient; {
		select {
		case URLScripts := <-chanURLScripts:
			for url, script := range URLScripts {
				finalURLScripts[url] = script
			}
		case <-chFinished:
			i++
		}
	}
	timeTrack(start, "callScraperHandler")
	return finalURLScripts
}
func callScraper(urls []string, callType string, chanURLScripts chan map[string]string, chFinished chan bool) {
	bodyMap := callScraperStruct{
		Type: callType,
		Urls: urls,
	}
	bodyJSON, _ := json.Marshal(bodyMap)
	client := &http.Client{}
	mutex.Lock()
	lambdaCount := lambdaCountUID
	lambdaCountUID++
	if lambdaCountUID >= 200 {
		lambdaCountUID = 0
	}
	mutex.Unlock()
	request, _ := http.NewRequest("POST", "https://1vzze2ned9.execute-api.us-east-1.amazonaws.com/default/test/go-scraper-"+strconv.Itoa(lambdaCount), bytes.NewBuffer(bodyJSON))
	response, _ := client.Do(request)
	body, _ := ioutil.ReadAll(response.Body)
	URLScripts := make(map[string]string)
	json.Unmarshal(body, &URLScripts)

	response.Body.Close()
	chanURLScripts <- URLScripts
	chFinished <- true
}

// --------------------------------- additional functions --------------------------------------
func after(value string, a string) string {
	// Get substring after a string.
	pos := strings.LastIndex(value, a)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(a)
	if adjustedPos >= len(value) {
		return ""
	}
	return value[adjustedPos:len(value)]
}
func before(value string, a string) string {
	pos := strings.Index(value, a)
	if pos == -1 {
		return ""
	}
	return value[0:pos]
}
func between(str string, start string, end string) (result string) {
	s := strings.Index(str, start)
	if s == -1 {
		return
	}
	s += len(start)
	e := strings.Index(str[s:], end)
	if e == -1 {
		return
	}
	return str[s : s+e]
}
func createNewError(name string, id int) error {
	return fmt.Errorf("user %q (id %d) not found", name, id)
}
func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
func checkViewsMultiplyer(stringData string) float64 {
	if strings.Contains(stringData, "천") {
		return 1000
	}
	if strings.Contains(stringData, "만") {
		return 10000
	}
	if strings.Contains(stringData, "K") {
		return 1000
	}
	if strings.Contains(stringData, "M") {
		return 1000000
	}
	if strings.Contains(stringData, "B") {
		return 1000000000
	}
	return 1
}
func checkForSocial(value string) (string, bool) {
	if strings.Contains(value, "facebook.com/groups") {
		return "FacebookGroup", true
	}
	if strings.Contains(value, "facebook.com/pages") {
		return "FacebookPage", true
	}
	if strings.Contains(value, "facebook") {
		return "Facebook", true
	}
	if strings.Contains(value, "twitch") {
		return "Twitch", true
	}
	if strings.Contains(value, "instagram") {
		return "Instagram", true
	}
	if strings.Contains(value, "twitter") {
		return "Twitter", true
	}
	return "", false
}
func isWithinYear(stringData string) bool {
	if strings.Contains(stringData, "day") {
		return true
	}
	if strings.Contains(stringData, "week") {
		return true
	}
	if strings.Contains(stringData, "month") {
		return true
	}
	if strings.Contains(stringData, "일") {
		return true
	}
	if strings.Contains(stringData, "주") {
		return true
	}
	return false
}
func removeButNumber(from string) int {
	if from == "" {
		return 0
	}
	reg, err := regexp.Compile("[^0-9]+")
	if err != nil {
		fmt.Printf("error: %v\n", err.Error())
		logger.Println(err.Error())
	}
	processedInt, err := strconv.Atoi(reg.ReplaceAllString(from, ""))
	if err != nil {
		fmt.Printf("error: %v\n", err.Error())
		logger.Println(err.Error())
	}
	return processedInt
}
func removeButFloat(from string) (returnFloat float64, err error) {
	reg, err := regexp.Compile("[^0-9.]+")
	if err != nil {
		return 0, err
	}
	processedString := reg.ReplaceAllString(from, "")
	resFloat, err := strconv.ParseFloat(processedString, 64)
	if err != nil {
		return 0, err

	}
	return resFloat, nil
}
func subscriberStringToInt(stringData string) int {
	if stringData == "" {
		return 0
	}
	var multiplier float64 = 1
	gotInt, err := removeButFloat(stringData)
	if err != nil {
		fmt.Printf("subscriberStringToInt error from string %v\n", stringData)
		fmt.Printf("error: %v\n", err.Error())
		logger.Printf("subscriberStringToInt error from string %v\n", stringData)
		logger.Println(err.Error())
		return 0
	}
	if strings.Contains(stringData, "천") {
		multiplier = 1000
	}
	if strings.Contains(stringData, "만") {
		multiplier = 10000
	}
	if strings.Contains(stringData, "억") {
		multiplier = 100000000
	}
	if strings.Contains(stringData, "K") {
		multiplier = 1000
	}
	if strings.Contains(stringData, "M") {
		multiplier = 1000000
	}
	if strings.Contains(stringData, "B") {
		multiplier = 1000000000
	}
	resInt := int(gotInt * multiplier)
	return resInt
}
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s\n", name, elapsed)
}
