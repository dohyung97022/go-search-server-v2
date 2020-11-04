package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

// --------------------------------- mutex var --------------------------------------
var (
	youtubeAPIRunOut = false
)

// --------------------------------- mutex var --------------------------------------
var (
	lambdaCountMax = 200
	lambdaCountUID = getInt.randomFromRange(0, lambdaCountMax)
	lambdaMutex    sync.Mutex
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
	// ----------------- check ytb api runout -----------------
	if youtubeAPIRunOut == true {
		fmt.Fprintf(w, "%s", "We have ran out of youtube api quotas. Please try again tomorrow.")
		return
	}
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
			if strings.Contains(err.Error(), "All ytb_api_key_quota is lower than 0.") {
				youtubeAPIRunOut = true
				return
			}
		}
		// ----------------- put data -----------------
		b.Reset()
		b.WriteString("INSERT INTO channels(channel, title, chan_url, last_update, chan_img, avr_views, ttl_views, subs, about) VALUES")
		for i, info := range intInfo {
			b.WriteString(aryWriter(
				"('", info.Channel, "','",
				strings.ReplaceAll(strings.ReplaceAll(info.Title, "'", "`"), "\\", ""), "','",
				info.ChanURL, "','",
				startTime.Format("2006-01-02 15:04:05"), "','",
				info.ChanImg, "','",
				strconv.Itoa(info.AvrViews), "','",
				strconv.Itoa(info.TTLViews), "','",
				strconv.Itoa(info.Subs), "','",
				strings.ReplaceAll(strings.ReplaceAll(info.About, "'", "`"), "\\", ""), "')"))
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
	getAll := queryOrDefaultStr("getall", "", r)
	if getAll != "true" {
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

// --------------------------------- Youtube API functions --------------------------------------
func getYoutubeAPIChannelsHandler(search string) (youtubeChannelIDAry []string, err error) {
	start := time.Now()
	youtubeChannelsMap := make(map[string]bool)

	APIRequestAmount := 100
	APIQuotaPerRequest := 1000
	APIQuotaPerSearch := APIRequestAmount * APIQuotaPerRequest

	ytbAPIKey, err := getYoutubeAPIKeyFromMysql(APIQuotaPerSearch)
	if err != nil {
		fmt.Printf("error : %v\n", err)
		logger.Println(err.Error())
		return nil, nil
	}

	pageToken := ""
	for i := 0; i < APIRequestAmount; i++ {
		youtubeChannels, newPageToken, err := getYoutubeAPIChannels(search, pageToken, ytbAPIKey)
		if err != nil {
			fmt.Printf("error : %v\n", err)
			logger.Println(err.Error())
			return nil, err
		}
		//결과가 나오지 않을 경우 다시 youtube api key를 받는다? 필요성이 있을까? 너무 복잡하지 않나?
		// if err != nil {
		// 	setYoutubeAPIKeyQuotaTo(ytbAPIKey, 0)
		// 	ytbAPIKey, err = getYoutubeAPIKeyFromMysql(APIQuotaPerSearch)
		// 	if err != nil {
		// 		fmt.Printf("error : %v\n", err)
		// 		logger.Println(err.Error())
		// 		return nil
		// 	}
		// }
		pageToken = newPageToken
		for _, youtubeChannel := range youtubeChannels {
			youtubeChannelsMap[youtubeChannel] = true
		}
	}
	timeTrack(start, "getYoutubeAPIChannelsHandler")
	return getStrAryFromStrBoolMap(youtubeChannelsMap), nil
}
func getYoutubeAPIChannels(search string, pageToken string, APIkey string) (youtubeChannels []string, nextPageToken string, err error) {
	response, err := http.Get("https://www.googleapis.com/youtube/v3/search?part=snippet&maxResults=50&type=channel&pageToken=" + pageToken + "&q=" + search + "&key=" + APIkey)
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
	//----------결과가 나오지 않았다면 json의 [error][errors][reason] = quotaExceeded ----------
	// 이 케이스의 경우 error를 주고 새롷은 api key로 다시 하게끔 한다?
	if len(youtubeChannels) == 0 {
		logger.Printf("check : no values was detected in script = \n%v\n", string(body))
	}
	return youtubeChannels, nextPageToken, nil
}
func getYoutubeAPIKeyFromMysql(APIQuotaPerSearch int) (ytbAPIKey string, err error) {
	v, err := msqlf.GetQuery(`
	SET @api_key = "";
	SET @quota = "";

	SELECT ytb_api_key, ytb_api_key_quota 
		INTO @api_key, @quota 
		FROM adiy.ytb_api_key ORDER BY ytb_api_key_quota DESC LIMIT 1;
		
	SELECT @api_key AS ytb_api_key, @quota AS ytb_api_key_quota;
	
	UPDATE adiy.ytb_api_key 
		SET ytb_api_key_quota = ytb_api_key_quota - ` + strconv.Itoa(APIQuotaPerSearch) + `
		WHERE ytb_api_key_quota > 0 AND ytb_api_key = @api_key;`)
	if err != nil {
		return
	}
	ytbAPIKey = v[0]["ytb_api_key"].(string)
	ytbAPIKeyQuota, err := strconv.Atoi(v[0]["ytb_api_key_quota"].(string))

	if ytbAPIKeyQuota <= 0 {
		err = errors.New("error : All ytb_api_key_quota is lower than 0. We need more api keys")
		return
	}
	return
}
func setYoutubeAPIKeyQuotaTo(APIKey string, quotaTo int) (err error) {
	var b strings.Builder
	b.WriteString(`
	UPDATE adiy.ytb_api_key 
		SET ytb_api_key_quota = ` + strconv.Itoa(quotaTo) + `
		WHERE ytb_api_key =` + APIKey)

	err = msqlf.ExecQuery(b.String())
	if err != nil {
		return err
	}
	return nil
}

// --------------------------------- scrape functions --------------------------------------
func scrape(search string) (channels []string, intInfo []info, err error) {
	search, _ = url.PathUnescape(search)
	search = strings.ReplaceAll(search, " ", "+")

	channels, err = getYoutubeAPIChannelsHandler(search)
	if err != nil {
		return nil, nil, err
	}

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
		Channel:    getStr.between(url, "https://www.youtube.com", "/videos"),
		UploadTime: "",
		AvrViews:   0,
	}
	viewsArray := strings.Split(s, "shortViewCountText\":{\"simpleText\":\"")
	// println(len(viewsArray))
	for i := range viewsArray {
		viewsArray[i] = getStr.before(viewsArray[i], "\"")
	}

	datesArray := strings.Split(s, "publishedTimeText\":{\"simpleText\":\"")
	// println(len(datesArray))
	for i := range datesArray {
		datesArray[i] = getStr.before(datesArray[i], "\"")
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
		if check.strWithinYear(datesArray[c]) == true {
			m := getFloat.fromViewUnitStr(viewsArray[c])
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
		Channel:    getStr.between(chanURL, "https://www.youtube.com", "/about"),
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
	if getStr.between(string(s), "<script>", "</script>") ==
		"var submitCallback = function(response) {document.getElementById('captcha-form').submit();};" {
		logger.Printf("Capcha has been detected. (crawlVideo) type 1 \n")
		fmt.Printf("error :%v\n", "Capcha has been detected. (crawlVideo) type 1")
	}
	// capcha type 2
	if strings.Contains(getStr.between(s, "<script src", "script>"), "https://www.google.com/recaptcha/api.js") == true {
		logger.Printf("Capcha has been detected. (crawlVideo) type 2\n")
		fmt.Printf("error :%v\n", "Capcha has been detected. (crawlVideo) type 2")
	}
	// capcha type 3
	if strings.Contains(getStr.between(s, "<script  src", "script>"), "https://www.google.com/recaptcha/api.js") == true {
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
		// fmt.Printf("Auto-generation has been detected\n")
	}
	if strings.Contains(s, "autoGenerated") {
		logger.Printf("Auto-generation has been detected\n")
		// fmt.Printf("Auto-generation has been detected\n")
	}

	// title
	title := getStr.between(s, "channelMetadataRenderer\":{\"title\":\"", "\"")
	storeInfo.Title += title
	//channel img
	img := getStr.between(s, "\"avatar\":{\"thumbnails\":[{\"url\":\"", "\"")
	storeInfo.ChanImg += img
	// total views
	views := getStr.between(s, "viewCountText", ",\"")
	storeInfo.TTLViews += getInt.fromStr(views)
	// abouts
	abouts := getStr.after(s, "\"channelMetadataRenderer\":{\"title\":\"")
	abouts = getStr.between(abouts, "description\":\"", "\",\"")
	storeInfo.About += abouts
	//email
	storeInfo.Links["Email"] = getStr.email(abouts)
	//subs
	subs := getStr.after(s, "subscriberCountText")
	subs = getStr.before(subs, "\"}")
	subs = getStr.after(subs, "\":\"")
	subs = strings.Replace(subs, "subscribers", "", 1)
	storeInfo.Subs = getInt.fromSubscriberUnit(subs)
	//links
	linksPre := getStr.between(s, "primaryLinks\":", "channelMetadataRenderer")
	linksArray := strings.Split(linksPre, "thumbnails")
	for val := range linksArray {
		link := getStr.after(linksArray[val], "urlEndpoint")
		link = getStr.between(link, "q=", "\"")
		if strings.Contains(link, "\\u0026") {
			link = getStr.before(link, "\\u0026")
		}
		decodedValue, _ := url.PathUnescape(link)
		if decodedValue != "" {
			//links url title
			title := getStr.between(linksArray[val], "title\":", "}}")
			title = getStr.between(title, ":\"", "\"")
			foundSocial, sucss := check.SocialInStr(decodedValue)
			if sucss {
				storeInfo.Links[foundSocial] = decodedValue
			}
		}
	}
	return
}

type callScraperStruct struct {
	//ioutil or goquery
	Type string
	//urls to scrape
	Urls []string
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

	lambdaMutex.Lock()
	lambdaCount := lambdaCountUID
	lambdaCountUID++
	if lambdaCountUID >= lambdaCountMax {
		lambdaCountUID = 0
	}
	lambdaMutex.Unlock()

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
type getStruct struct{}

var get getStruct

type getStrStruct struct{}

var getStr getStrStruct

func (getStrStruct) after(value string, after string) string {
	pos := strings.LastIndex(value, after)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(after)
	if adjustedPos >= len(value) {
		return ""
	}
	return value[adjustedPos:len(value)]
}
func (getStrStruct) before(value string, before string) string {
	pos := strings.Index(value, before)
	if pos == -1 {
		return ""
	}
	return value[0:pos]
}
func (getStrStruct) between(value string, start string, end string) string {
	s := strings.Index(value, start)
	if s == -1 {
		return ""
	}
	s += len(start)
	e := strings.Index(value[s:], end)
	if e == -1 {
		return ""
	}
	return value[s : s+e]
}
func (getStrStruct) email(value string) string {
	re := regexp.MustCompile(`[a-zA-Z0-9]+@[a-zA-Z0-9\.]+\.[a-zA-Z0-9]+`)
	email := re.FindString(value)
	return email
}

type getIntStruct struct{}

var getInt getIntStruct

func (getIntStruct) randomFromRange(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}
func (getIntStruct) fromStr(from string) int {
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
func (getIntStruct) fromSubscriberUnit(subscriberUnit string) int {
	if subscriberUnit == "" {
		return 0
	}
	var multiplier float64 = 1
	gotInt, err := getFloat.fromStr(subscriberUnit)
	if err != nil {
		fmt.Printf("subscriberStringToInt error from string %v\n", subscriberUnit)
		fmt.Printf("error: %v\n", err.Error())
		logger.Printf("subscriberStringToInt error from string %v\n", subscriberUnit)
		logger.Println(err.Error())
		return 0
	}
	if strings.Contains(subscriberUnit, "천") {
		multiplier = 1000
	}
	if strings.Contains(subscriberUnit, "만") {
		multiplier = 10000
	}
	if strings.Contains(subscriberUnit, "억") {
		multiplier = 100000000
	}
	if strings.Contains(subscriberUnit, "K") {
		multiplier = 1000
	}
	if strings.Contains(subscriberUnit, "M") {
		multiplier = 1000000
	}
	if strings.Contains(subscriberUnit, "B") {
		multiplier = 1000000000
	}
	resInt := int(gotInt * multiplier)
	return resInt
}

type getFloatStruct struct{}

var getFloat getFloatStruct

func (getFloatStruct) fromViewUnitStr(viewUnit string) float64 {
	if strings.Contains(viewUnit, "천") {
		return 1000
	}
	if strings.Contains(viewUnit, "만") {
		return 10000
	}
	if strings.Contains(viewUnit, "K") {
		return 1000
	}
	if strings.Contains(viewUnit, "M") {
		return 1000000
	}
	if strings.Contains(viewUnit, "B") {
		return 1000000000
	}
	return 1
}
func (getFloatStruct) fromStr(from string) (float64, error) {
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

type checkStruct struct{}

var check checkStruct

func (checkStruct) strArayContains(strArry []string, contains string) bool {
	for _, a := range strArry {
		if a == contains {
			return true
		}
	}
	return false
}
func (checkStruct) SocialInStr(value string) (string, bool) {
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
func (checkStruct) strWithinYear(stringData string) bool {
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

func aryWriter(strAry ...string) string {
	var b strings.Builder
	for _, str := range strAry {
		b.WriteString(str)
	}
	return b.String()
}
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s\n", name, elapsed)
}
func getStrAryFromStrBoolMap(strBoolMap map[string]bool) (strAry []string) {
	strAry = make([]string, len(strBoolMap))
	for str := range strBoolMap {
		strAry = append(strAry, str)
	}
	return strAry
}
