package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
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
	lambdaCountUID = createRandomFromRange(0, 200)
	mutex          sync.Mutex
)

// --------------------------------- logger var --------------------------------------
var (
	loggerFile, _ = os.OpenFile("err.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logger        = log.New(loggerFile, "Log", log.LstdFlags|log.Lshortfile)
)

// --------------------------------- server functions --------------------------------------
//http://ec2-54-161-234-228.compute-1.amazonaws.com:3000/search?search=
// http://localhost:3000/search?search=
func main() {
	err := msqlf.Init("dohyung97022", "9347314da!", "adiy-db.cxdzwqqcqoib.us-east-1.rds.amazonaws.com", 3306, "adiy")
	if err != nil {
		fmt.Printf("error : %v\n", err)
		logger.Println(err.Error())
		return
	}
	fmt.Println("server is up and running")
	http.HandleFunc("/search", handler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}
func handler(w http.ResponseWriter, r *http.Request) {
	// ----------------- header -----------------
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-type", "application/json; charset=UTF-8")
	// ----------------- execution time -----------------
	fmt.Println("request on 3000 (search)")
	startTime := time.Now()
	defer func() {
		fmt.Printf("Binomial took %v\n", time.Since(startTime))
	}()
	// ----------------- search parameters -----------------
	var b strings.Builder
	search := queryOrDefaultStr("search", "", r)
	if search == "" {
		log.Println("Url Param 'search' is missing")
		fmt.Fprintf(w, "You are missing the 'search' param!\n")
		return
	}

	// ----------------- need scraping? -----------------
	v, err := msqlf.GetDataOfWhere("search", []string{"last_update", "srch_id"},
		[]msqlf.Where{msqlf.Where{A: "query", IS: "=", B: search}})
	if err != nil {
		fmt.Printf("error : %v\n", err)
		logger.Println(err.Error())
		return
	}
	// ----------------- varify needRef and sechID -----------------
	needRef := false
	var srchID int
	if len(v) == 0 {
		// search table has no data of query
		needRef = true
		b.WriteString(aryWriter("INSERT INTO search(query, last_update) VALUES('", search, "','", startTime.Format("2006-01-02 15:04:05"), "'); SELECT last_insert_id();"))

		v, err := msqlf.GetQuery(b.String())
		if err != nil {
			fmt.Printf("error : %v\n", err)
			logger.Println(err.Error())
			return
		}
		srchID, err = strconv.Atoi(v[0]["last_insert_id()"].(string))
		if err != nil {
			fmt.Printf("error : %v\n", err)
			logger.Println(err.Error())
			return
		}

	} else {
		// ----------------- search has data of query -----------------
		t, err := time.Parse("2006-01-02 15:04:05", v[0]["last_update"].(string))
		if err != nil {
			fmt.Printf("error : %v\n", err)
			logger.Println(err.Error())
			return
		}
		// ----------------- outdated -----------------
		if t.Before(startTime.AddDate(0, 0, -2)) {
			needRef = true
			srchID, err = strconv.Atoi(v[0]["srch_id"].(string))
			if err != nil {
				fmt.Printf("error : %v\n", err)
				logger.Println(err.Error())
				return
			}
			// ----------------- update last_update -----------------
			b.Reset()
			b.WriteString(aryWriter("UPDATE search SET last_update = '", startTime.Format("2006-01-02 15:04:05"), "' WHERE srch_id = ", strconv.Itoa(srchID)))
			err = msqlf.ExecQuery(b.String())
			if err != nil {
				fmt.Printf("error : %v\n", err)
				logger.Println(err.Error())
				logger.Printf("error : SQL Query : %s", b.String())
				return
			}
		}
	}

	// channelBools := make(map[string]bool)
	if needRef {
		// ----------------- scrape, put or update data -----------------
		_, intInfo, err := scrape(search)
		if err != nil {
			fmt.Printf("error : %v\n", err)
			logger.Println(err.Error())
			return
		}
		// ----------------- put data -----------------
		b.Reset()
		b.WriteString("INSERT INTO channels(channel, title, chan_url, last_update, chan_img, avr_views, ttl_views, subs, about) VALUES")
		for i, info := range intInfo {
			b.WriteString(aryWriter(
				"('", info.Channel, "','",
				strings.ReplaceAll(info.Title, "'", "`"), "','",
				info.ChanURL, "','",
				startTime.Format("2006-01-02 15:04:05"), "','",
				info.ChanImg, "','",
				strconv.Itoa(info.AvrViews), "','",
				strconv.Itoa(info.TTLViews), "','",
				strconv.Itoa(info.Subs), "','",
				strings.ReplaceAll(info.About, "'", "`"), "')"))
			if i+1 == len(intInfo) {
				break
			}
			b.WriteString(",")
		}
		// ----------------- update data -----------------
		b.WriteString(" AS dpc ON DUPLICATE KEY UPDATE title=dpc.title, chan_url=dpc.chan_url, last_update=dpc.last_update, chan_img=dpc.chan_img, avr_views=dpc.avr_views, ttl_views=dpc.ttl_views, subs=dpc.subs, about=dpc.about;")
		// ----------------- exec query -----------------
		err = msqlf.ExecQuery(b.String())
		if err != nil {
			fmt.Printf("error : %v\n", err)
			logger.Println(err.Error())
			logger.Printf("error : SQL Query : %s", b.String())
			return
		}
		// ----------------- put contacts -----------------
		b.Reset()
		b.WriteString("INSERT INTO contacts(channel, facebook, facebook_group, facebook_page, twitch, instagram, twitter, email) VALUES")
		for i, info := range intInfo {
			// FacebookGroup
			b.WriteString(aryWriter(
				"('", info.Channel, "','",
				info.Links["Facebook"], "','",
				info.Links["FacebookGroup"], "','",
				info.Links["FacebookPage"], "','",
				info.Links["Twitch"], "','",
				info.Links["Instagram"], "','",
				info.Links["Twitter"], "','",
				info.Links["Email"], "')"))
			if i+1 == len(intInfo) {
				break
			}
			b.WriteString(",")
		}
		// ----------------- update contacts -----------------
		b.WriteString(` AS dpc ON DUPLICATE KEY UPDATE 
		facebook=(CASE WHEN dpc.facebook='' THEN contacts.facebook ELSE dpc.facebook END),
		 facebook_group=(CASE WHEN dpc.facebook_group='' THEN contacts.facebook_group ELSE dpc.facebook_group END),
		  facebook_page=(CASE WHEN dpc.facebook_page='' THEN contacts.facebook_page ELSE dpc.facebook_page END),
		   twitch=(CASE WHEN dpc.twitch='' THEN contacts.twitch ELSE dpc.twitch END),
			instagram=(CASE WHEN dpc.instagram='' THEN contacts.instagram ELSE dpc.instagram END),
			 twitter=(CASE WHEN dpc.twitter='' THEN contacts.twitter ELSE dpc.twitter END),
			  email=(CASE WHEN dpc.email='' THEN contacts.email ELSE dpc.email END);`)

		// ----------------- exec query -----------------
		err = msqlf.ExecQuery(b.String())
		if err != nil {
			fmt.Printf("error : %v\n", err)
			logger.Println(err.Error())
			logger.Printf("error : SQL Query : %s", b.String())
			return
		}
		// ----------------- one to many relations with srch_id -----------------
		b.Reset()
		b.WriteString("INSERT IGNORE INTO search_channels(srch_id, channel) VALUES")
		for i, info := range intInfo {
			b.WriteString(aryWriter("(", strconv.Itoa(srchID), ",'", info.Channel, "')"))
			if i+1 == len(intInfo) {
				b.WriteString(";")
				break
			}
			b.WriteString(",")
		}
		err = msqlf.ExecQuery(b.String())
		if err != nil {
			fmt.Printf("error : %v\n", err)
			logger.Println(err.Error())
			logger.Printf("error : SQL Query : %s", b.String())
			return
		}
	}
	// ----------------- condition parameters -----------------
	avMin := queryOrDefaultStr("avmin", "", r)
	avMax := queryOrDefaultStr("avmax", "", r)
	sbMin := queryOrDefaultStr("sbmin", "", r)
	sbMax := queryOrDefaultStr("sbmax", "", r)
	// ----------------- fetch data -----------------
	b.Reset()
	b.WriteString(aryWriter("SELECT * FROM channels_views WHERE query = '", search, "' "))
	if avMin != "" {
		b.WriteString(aryWriter("AND avr_views >= '", avMin, "' "))
	}
	if avMax != "" {
		b.WriteString(aryWriter("AND avr_views <= '", avMax, "' "))
	}
	if sbMin != "" {
		b.WriteString(aryWriter("AND subs >= '", sbMin, "' "))
	}
	if sbMax != "" {
		b.WriteString(aryWriter("AND subs <= '", sbMax, "' "))
	}
	// ----------------- page parameter -----------------
	pageInt, err := strconv.Atoi(queryOrDefaultStr("page", "1", r))
	amountInPage := 20
	if err != nil {
		fmt.Printf("error : %v\n", err)
		logger.Println(err.Error())
		return
	}
	// ----------------- getallpage parameter -----------------
	getAllPage := queryOrDefaultStr("getallpage", "", r)
	if getAllPage != "true" {
		b.WriteString(aryWriter("LIMIT ", strconv.Itoa((pageInt-1)*amountInPage), ", ", strconv.Itoa(amountInPage), " "))
	}
	v, err = msqlf.GetQuery(b.String())
	if err != nil {
		fmt.Printf("error : %v\n", err)
		logger.Println(err.Error())
		return
	}
	bodyJSON, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("error :%v\n", err)
		logger.Println(err.Error())
		return
	}
	fmt.Fprintf(w, "%s", bodyJSON)

	//데이터가 존재했었다. 이전의 업데이트가 부족한 체널들 scrape?
	// if srchID != -1 {
	// 	go scrapeOutdated(search, channelBools, intInfo)
	// }
}
func queryOrDefaultStr(query string, def string, r *http.Request) string {
	params, ok := r.URL.Query()[query]
	if !ok || len(params) == 0 {
		return def
	}
	return params[0]
}

// --------------------------------- scrape functions --------------------------------------
func scrape(search string) (channels []string, intInfo []info, err error) {
	search, _ = url.PathUnescape(search)
	search = strings.ReplaceAll(search, " ", "+")

	//youtube api key를 kubernetes에서 공용으로 load balancing하는 방법을 고안하기
	// channels, nextPageToken, err := getYoutubeAPIChannels(search, "AIzaSyDIc53xLxBg4W6etfMhzuf9nqdbmsqsKOc")
	// if err != nil {
	// 	return nil, nil, err
	// }
	aboutUrlsArray := []string{}
	for _, channel := range channels {
		aboutUrlsArray = append(aboutUrlsArray, "https://www.youtube.com/channel/"+channel+"/about")
	}
	videosUrlsArray := []string{}
	for _, channel := range channels {
		videosUrlsArray = append(videosUrlsArray, "https://www.youtube.com/channel/"+channel+"/videos")
	}
	chAbout := make(chan map[string]string)
	chVideos := make(chan map[string]string)
	go func() { chAbout <- callScraperHandler(aboutUrlsArray, "goquery") }()
	go func() { chVideos <- callScraperHandler(videosUrlsArray, "goquery") }()

	URLScriptAbout := <-chAbout
	URLScriptVideos := <-chVideos

	logger.Printf("Url length : %v", len(URLScriptAbout))

	chanInfo := findInfoHandler(URLScriptAbout)
	chanVideosInfo := findVideosInfoHandler(URLScriptVideos)

	for url, info := range chanInfo {
		info.AvrViews = chanVideosInfo[url].AvrViews
		info.UploadTime = chanVideosInfo[url].UploadTime
		intInfo = append(intInfo, info)
	}

	return channels, intInfo, nil
}

func getYoutubeAPIChannels(search string, APIkey string) (youtubeChannels []string, nextPageToken string, err error) {
	response, err := http.Get("https://www.googleapis.com/youtube/v3/search?part=snippet&maxResults=50&type=channel&q=" + search + "&key=" + APIkey)
	if err != nil {
		log.Fatal(err)
		return nil, "", err
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	//----------json 의 [nextPageToken]----------
	ytbAPIResStrStr := make(map[string]string)
	json.Unmarshal(body, &ytbAPIResStrStr)
	nextPageToken = ytbAPIResStrStr["nextPageToken"]
	//----------json 의 [items][id][channelId]----------
	ytbAPIResStrInterfAry := make(map[string][]interface{})
	json.Unmarshal(body, &ytbAPIResStrInterfAry)
	for _, i := range ytbAPIResStrInterfAry["items"] {
		items := i.(map[string]interface{})
		id := items["id"].(map[string]interface{})
		youtubeChannels = append(youtubeChannels, id["channelId"].(string))
	}
	return youtubeChannels, nextPageToken, nil
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
	TTLViews   int
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
		TTLViews:   0,
		AvrViews:   0,
		UploadTime: "",
		Links: map[string]string{
			"FacebookGroup": "",
			"FacebookPage":  "",
			"Facebook":      "",
			"Twitch":        "",
			"Instagram":     "",
			"Twitter":       "",
			"Email":         "",
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
	storeInfo.TTLViews += removeButNumber(views)
	// abouts
	abouts := after(s, "\"channelMetadataRenderer\":{\"title\":\"")
	abouts = between(abouts, "description\":\"", "\",\"")
	storeInfo.About += abouts
	//email
	storeInfo.Links["Email"] = getEmail(abouts)
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
	//devider가 작을수록 더 scraper가 많이 분산됩니다. devider는 lambda마다의 과부화
	devider := 2
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
func aryWriter(strAry ...string) string {
	var b strings.Builder
	for _, str := range strAry {
		b.WriteString(str)
	}
	return b.String()
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
func createRandomFromRange(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
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
func getEmail(text string) string {
	re := regexp.MustCompile(`[a-zA-Z0-9]+@[a-zA-Z0-9\.]+\.[a-zA-Z0-9]+`)
	match := re.FindString(text)
	return match
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
