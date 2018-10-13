//----------------
//Func  :
//Author: xjh
//Date  : 2018/
//Note  :
//----------------
package googletrends

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/bitly/go-simplejson"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	GET_METHOD             = "GET"
	POST_METHOD            = "POST"
	COOKIE_URL             = "https://trends.google.com"
	GENERAL_URL            = "https://trends.google.com/trends/api/explore"
	INTEREST_OVER_TIME_URL = "https://trends.google.com/trends/api/widgetdata/multiline"
)

func NewTrendReq(hl string, tz int, geo, timeframe, proxies string, kw_list []string) *TrendReq {
	ret := &TrendReq{
		tz:                        tz,
		hl:                        hl,
		geo:                       geo,
		timeframe:                 timeframe,
		kw_list:                   kw_list,
		proxy:                     func(_ *http.Request) (*url.URL, error) { return url.Parse(proxies) },
		token_payload:             map[string]interface{}{},
		interest_over_time_widget: nil,
	}

	if len(proxies) > 0 {
		ret.proxy = func(_ *http.Request) (*url.URL, error) { return url.Parse(proxies) }
	} else {
		ret.proxy = nil
	}

	client := &http.Client{Transport: &http.Transport{Proxy: ret.proxy}}
	resp, err := client.Get(COOKIE_URL)
	if err != nil {
		logs.Warn("request:%s err:%v", COOKIE_URL, err)
		return nil
	}
	ret.cookies = resp.Cookies()
	logs.Info("new trendreq:%v", ret)
	ret.build_payload()
	return ret
}

type TrendReq struct {
	tz                        int                                     //480
	hl                        string                                  //"en-US"
	geo                       string                                  //"CN"
	timeframe                 string                                  //"today 12-m"
	kw_list                   []string                                //["vim", "world"]
	proxy                     func(_ *http.Request) (*url.URL, error) //"http": "http://192.168.0.1:8888"
	cookies                   []*http.Cookie                          //{'NID': '140=7TVi6hyP5vTzUjEs-1eXVRRpR_xH_6cfzZqZsemrPKVz94jffdlsX_4HOvuAfNKBpOH_NmYUVMmKX_zXzs5Amn8JDjiSWu9YzcZk0e3m1JqElI5gHe2pQYylF2KYiql4'}
	token_payload             map[string]interface{}                  //dict
	interest_over_time_widget *simplejson.Json                        //json
}

func (self *TrendReq) build_payload() {
	self.token_payload["hl"] = self.hl
	self.token_payload["tz"] = self.tz
	comparisonItem := []map[string]interface{}{}
	for _, kw := range self.kw_list {
		keyword_payload := map[string]interface{}{"keyword": kw, "time": self.timeframe, "geo": self.geo}
		comparisonItem = append(comparisonItem, keyword_payload)
	}
	cpi, _ := json.Marshal(map[string]interface{}{"comparisonItem": comparisonItem, "category": 0, "property": ""})
	self.token_payload["req"] = string(cpi)
	logs.Info("token_payload:%v", self.token_payload)
	self._tokens()
}

func (self *TrendReq) _tokens() {
	widgets := self._get_data(GENERAL_URL, 4, self.token_payload).Get("widgets")
	ws, err := widgets.Array()
	if err != nil {
		logs.Error("_tokens widgets err:%v", err)
	}
	for idx, _ := range ws {
		wj := widgets.GetIndex(idx)
		wid, _ := wj.Get("id").String()
		if wid == "TIMESERIES" {
			self.interest_over_time_widget = wj
			break
		}
	}
}

func (self *TrendReq) _get_data(url string, trim_chars int, params map[string]interface{}) *simplejson.Json {
	client := &http.Client{Transport: &http.Transport{Proxy: self.proxy}}
	req, _ := http.NewRequest(GET_METHOD, url, nil)
	for _, c := range self.cookies {
		req.AddCookie(c)
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, fmt.Sprintf("%v", v))
	}
	req.URL.RawQuery = q.Encode()
	//logs.Info("get_data url:%s", req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		logs.Warn("request:%s err:%v", url, err)
		return nil
	}
	ct := resp.Header.Get("Content-Type")
	if strings.IndexAny(ct, "application/json") >= 0 || strings.IndexAny(ct, "application/javascript") >= 0 || strings.IndexAny(ct, "text/javascript") >= 0 {
		s, _ := ioutil.ReadAll(resp.Body)
		js, err := simplejson.NewJson([]byte(s[trim_chars:]))
		if err != nil {
			logs.Info("get_data resp:%s", string(s))
			logs.Error("get_data json err:%v", err)
		}
		return js
	} else {
		logs.Error("get_data url:%s code:%d resp:%s", url, resp.StatusCode, resp.Body)
	}
	return nil
}

func (self *TrendReq) InterestOverTime() *simplejson.Json {
	req, err := self.interest_over_time_widget.Get("request").Encode()
	if err != nil {
		logs.Error("interest_over_time request err:%v data:%v", err, self.interest_over_time_widget.Get("request"))
		return nil
	}
	token, err := self.interest_over_time_widget.Get("token").String()
	if err != nil {
		logs.Error("interest_over_time token err:%v data:%v", err, self.interest_over_time_widget.Get("token"))
		return nil
	}
	over_time_payload := map[string]interface{}{
		"req":   string(req),
		"token": token,
		"tz":    self.tz,
	}
	js := self._get_data(INTEREST_OVER_TIME_URL, 5, over_time_payload)
	return js
}

var Month = map[string]int{
	"jan": 1,  //January
	"feb": 2,  //February
	"mar": 3,  //March
	"apr": 4,  //April
	"may": 5,  //May
	"jun": 6,  //June
	"jul": 7,  //July
	"aug": 8,  //August
	"sep": 9,  //September
	"oct": 10, //October
	"nov": 11, //November
	"dec": 12, //December
}

type WeekData struct {
	StartY    int  //起始年
	StartM    int  //起始月
	StartD    int  //起始日
	EndY      int  //结束年
	EndM      int  //结束月
	EndD      int  //结束日
	Value     int  //值
	IsPartial bool //是否是预测
}

//解析数据
func ParseInterestOverTime(js *simplejson.Json) []*WeekData {
	//测试数据
	//data := "{\"default\":{\"averages\":[],\"timelineData\":[{\"formattedAxisTime\":\"Oct 15, 2017\",\"formattedTime\":\"Oct 15 - Oct 21 2017\",\"formattedValue\":[\"81\"],\"hasData\":[true],\"time\":\"1508025600\",\"value\":[81]},{\"formattedAxisTime\":\"Oct 22, 2017\",\"formattedTime\":\"Oct 22 - Oct 28 2017\",\"formattedValue\":[\"76\"],\"hasData\":[true],\"time\":\"1508630400\",\"value\":[76]},{\"formattedAxisTime\":\"Oct 29, 2017\",\"formattedTime\":\"Oct 29 - Nov 4 2017\",\"formattedValue\":[\"83\"],\"hasData\":[true],\"time\":\"1509235200\",\"value\":[83]},{\"formattedAxisTime\":\"Nov 5, 2017\",\"formattedTime\":\"Nov 5 - Nov 11 2017\",\"formattedValue\":[\"84\"],\"hasData\":[true],\"time\":\"1509840000\",\"value\":[84]},{\"formattedAxisTime\":\"Nov 12, 2017\",\"formattedTime\":\"Nov 12 - Nov 18 2017\",\"formattedValue\":[\"82\"],\"hasData\":[true],\"time\":\"1510444800\",\"value\":[82]},{\"formattedAxisTime\":\"Nov 19, 2017\",\"formattedTime\":\"Nov 19 - Nov 25 2017\",\"formattedValue\":[\"76\"],\"hasData\":[true],\"time\":\"1511049600\",\"value\":[76]},{\"formattedAxisTime\":\"Nov 26, 2017\",\"formattedTime\":\"Nov 26 - Dec 2 2017\",\"formattedValue\":[\"84\"],\"hasData\":[true],\"time\":\"1511654400\",\"value\":[84]},{\"formattedAxisTime\":\"Dec 3, 2017\",\"formattedTime\":\"Dec 3 - Dec 9 2017\",\"formattedValue\":[\"82\"],\"hasData\":[true],\"time\":\"1512259200\",\"value\":[82]},{\"formattedAxisTime\":\"Dec 10, 2017\",\"formattedTime\":\"Dec 10 - Dec 16 2017\",\"formattedValue\":[\"80\"],\"hasData\":[true],\"time\":\"1512864000\",\"value\":[80]},{\"formattedAxisTime\":\"Dec 17, 2017\",\"formattedTime\":\"Dec 17 - Dec 23 2017\",\"formattedValue\":[\"84\"],\"hasData\":[true],\"time\":\"1513468800\",\"value\":[84]},{\"formattedAxisTime\":\"Dec 24, 2017\",\"formattedTime\":\"Dec 24 - Dec 30 2017\",\"formattedValue\":[\"85\"],\"hasData\":[true],\"time\":\"1514073600\",\"value\":[85]},{\"formattedAxisTime\":\"Dec 31, 2017\",\"formattedTime\":\"Dec 31 2017 - Jan 6 2018\",\"formattedValue\":[\"77\"],\"hasData\":[true],\"time\":\"1514678400\",\"value\":[77]},{\"formattedAxisTime\":\"Jan 7, 2018\",\"formattedTime\":\"Jan 7 - Jan 13 2018\",\"formattedValue\":[\"76\"],\"hasData\":[true],\"time\":\"1515283200\",\"value\":[76]},{\"formattedAxisTime\":\"Jan 14, 2018\",\"formattedTime\":\"Jan 14 - Jan 20 2018\",\"formattedValue\":[\"85\"],\"hasData\":[true],\"time\":\"1515888000\",\"value\":[85]},{\"formattedAxisTime\":\"Jan 21, 2018\",\"formattedTime\":\"Jan 21 - Jan 27 2018\",\"formattedValue\":[\"74\"],\"hasData\":[true],\"time\":\"1516492800\",\"value\":[74]},{\"formattedAxisTime\":\"Jan 28, 2018\",\"formattedTime\":\"Jan 28 - Feb 3 2018\",\"formattedValue\":[\"80\"],\"hasData\":[true],\"time\":\"1517097600\",\"value\":[80]},{\"formattedAxisTime\":\"Feb 4, 2018\",\"formattedTime\":\"Feb 4 - Feb 10 2018\",\"formattedValue\":[\"72\"],\"hasData\":[true],\"time\":\"1517702400\",\"value\":[72]},{\"formattedAxisTime\":\"Feb 11, 2018\",\"formattedTime\":\"Feb 11 - Feb 17 2018\",\"formattedValue\":[\"58\"],\"hasData\":[true],\"time\":\"1518307200\",\"value\":[58]},{\"formattedAxisTime\":\"Feb 18, 2018\",\"formattedTime\":\"Feb 18 - Feb 24 2018\",\"formattedValue\":[\"72\"],\"hasData\":[true],\"time\":\"1518912000\",\"value\":[72]},{\"formattedAxisTime\":\"Feb 25, 2018\",\"formattedTime\":\"Feb 25 - Mar 3 2018\",\"formattedValue\":[\"86\"],\"hasData\":[true],\"time\":\"1519516800\",\"value\":[86]},{\"formattedAxisTime\":\"Mar 4, 2018\",\"formattedTime\":\"Mar 4 - Mar 10 2018\",\"formattedValue\":[\"82\"],\"hasData\":[true],\"time\":\"1520121600\",\"value\":[82]},{\"formattedAxisTime\":\"Mar 11, 2018\",\"formattedTime\":\"Mar 11 - Mar 17 2018\",\"formattedValue\":[\"91\"],\"hasData\":[true],\"time\":\"1520726400\",\"value\":[91]},{\"formattedAxisTime\":\"Mar 18, 2018\",\"formattedTime\":\"Mar 18 - Mar 24 2018\",\"formattedValue\":[\"85\"],\"hasData\":[true],\"time\":\"1521331200\",\"value\":[85]},{\"formattedAxisTime\":\"Mar 25, 2018\",\"formattedTime\":\"Mar 25 - Mar 31 2018\",\"formattedValue\":[\"97\"],\"hasData\":[true],\"time\":\"1521936000\",\"value\":[97]},{\"formattedAxisTime\":\"Apr 1, 2018\",\"formattedTime\":\"Apr 1 - Apr 7 2018\",\"formattedValue\":[\"85\"],\"hasData\":[true],\"time\":\"1522540800\",\"value\":[85]},{\"formattedAxisTime\":\"Apr 8, 2018\",\"formattedTime\":\"Apr 8 - Apr 14 2018\",\"formattedValue\":[\"86\"],\"hasData\":[true],\"time\":\"1523145600\",\"value\":[86]},{\"formattedAxisTime\":\"Apr 15, 2018\",\"formattedTime\":\"Apr 15 - Apr 21 2018\",\"formattedValue\":[\"88\"],\"hasData\":[true],\"time\":\"1523750400\",\"value\":[88]},{\"formattedAxisTime\":\"Apr 22, 2018\",\"formattedTime\":\"Apr 22 - Apr 28 2018\",\"formattedValue\":[\"88\"],\"hasData\":[true],\"time\":\"1524355200\",\"value\":[88]},{\"formattedAxisTime\":\"Apr 29, 2018\",\"formattedTime\":\"Apr 29 - May 5 2018\",\"formattedValue\":[\"84\"],\"hasData\":[true],\"time\":\"1524960000\",\"value\":[84]},{\"formattedAxisTime\":\"May 6, 2018\",\"formattedTime\":\"May 6 - May 12 2018\",\"formattedValue\":[\"90\"],\"hasData\":[true],\"time\":\"1525564800\",\"value\":[90]},{\"formattedAxisTime\":\"May 13, 2018\",\"formattedTime\":\"May 13 - May 19 2018\",\"formattedValue\":[\"100\"],\"hasData\":[true],\"time\":\"1526169600\",\"value\":[100]},{\"formattedAxisTime\":\"May 20, 2018\",\"formattedTime\":\"May 20 - May 26 2018\",\"formattedValue\":[\"94\"],\"hasData\":[true],\"time\":\"1526774400\",\"value\":[94]},{\"formattedAxisTime\":\"May 27, 2018\",\"formattedTime\":\"May 27 - Jun 2 2018\",\"formattedValue\":[\"87\"],\"hasData\":[true],\"time\":\"1527379200\",\"value\":[87]},{\"formattedAxisTime\":\"Jun 3, 2018\",\"formattedTime\":\"Jun 3 - Jun 9 2018\",\"formattedValue\":[\"89\"],\"hasData\":[true],\"time\":\"1527984000\",\"value\":[89]},{\"formattedAxisTime\":\"Jun 10, 2018\",\"formattedTime\":\"Jun 10 - Jun 16 2018\",\"formattedValue\":[\"89\"],\"hasData\":[true],\"time\":\"1528588800\",\"value\":[89]},{\"formattedAxisTime\":\"Jun 17, 2018\",\"formattedTime\":\"Jun 17 - Jun 23 2018\",\"formattedValue\":[\"86\"],\"hasData\":[true],\"time\":\"1529193600\",\"value\":[86]},{\"formattedAxisTime\":\"Jun 24, 2018\",\"formattedTime\":\"Jun 24 - Jun 30 2018\",\"formattedValue\":[\"85\"],\"hasData\":[true],\"time\":\"1529798400\",\"value\":[85]},{\"formattedAxisTime\":\"Jul 1, 2018\",\"formattedTime\":\"Jul 1 - Jul 7 2018\",\"formattedValue\":[\"96\"],\"hasData\":[true],\"time\":\"1530403200\",\"value\":[96]},{\"formattedAxisTime\":\"Jul 8, 2018\",\"formattedTime\":\"Jul 8 - Jul 14 2018\",\"formattedValue\":[\"98\"],\"hasData\":[true],\"time\":\"1531008000\",\"value\":[98]},{\"formattedAxisTime\":\"Jul 15, 2018\",\"formattedTime\":\"Jul 15 - Jul 21 2018\",\"formattedValue\":[\"93\"],\"hasData\":[true],\"time\":\"1531612800\",\"value\":[93]},{\"formattedAxisTime\":\"Jul 22, 2018\",\"formattedTime\":\"Jul 22 - Jul 28 2018\",\"formattedValue\":[\"96\"],\"hasData\":[true],\"time\":\"1532217600\",\"value\":[96]},{\"formattedAxisTime\":\"Jul 29, 2018\",\"formattedTime\":\"Jul 29 - Aug 4 2018\",\"formattedValue\":[\"86\"],\"hasData\":[true],\"time\":\"1532822400\",\"value\":[86]},{\"formattedAxisTime\":\"Aug 5, 2018\",\"formattedTime\":\"Aug 5 - Aug 11 2018\",\"formattedValue\":[\"88\"],\"hasData\":[true],\"time\":\"1533427200\",\"value\":[88]},{\"formattedAxisTime\":\"Aug 12, 2018\",\"formattedTime\":\"Aug 12 - Aug 18 2018\",\"formattedValue\":[\"84\"],\"hasData\":[true],\"time\":\"1534032000\",\"value\":[84]},{\"formattedAxisTime\":\"Aug 19, 2018\",\"formattedTime\":\"Aug 19 - Aug 25 2018\",\"formattedValue\":[\"90\"],\"hasData\":[true],\"time\":\"1534636800\",\"value\":[90]},{\"formattedAxisTime\":\"Aug 26, 2018\",\"formattedTime\":\"Aug 26 - Sep 1 2018\",\"formattedValue\":[\"94\"],\"hasData\":[true],\"time\":\"1535241600\",\"value\":[94]},{\"formattedAxisTime\":\"Sep 2, 2018\",\"formattedTime\":\"Sep 2 - Sep 8 2018\",\"formattedValue\":[\"94\"],\"hasData\":[true],\"time\":\"1535846400\",\"value\":[94]},{\"formattedAxisTime\":\"Sep 9, 2018\",\"formattedTime\":\"Sep 9 - Sep 15 2018\",\"formattedValue\":[\"90\"],\"hasData\":[true],\"time\":\"1536451200\",\"value\":[90]},{\"formattedAxisTime\":\"Sep 16, 2018\",\"formattedTime\":\"Sep 16 - Sep 22 2018\",\"formattedValue\":[\"87\"],\"hasData\":[true],\"time\":\"1537056000\",\"value\":[87]},{\"formattedAxisTime\":\"Sep 23, 2018\",\"formattedTime\":\"Sep 23 - Sep 29 2018\",\"formattedValue\":[\"94\"],\"hasData\":[true],\"time\":\"1537660800\",\"value\":[94]},{\"formattedAxisTime\":\"Sep 30, 2018\",\"formattedTime\":\"Sep 30 - Oct 6 2018\",\"formattedValue\":[\"82\"],\"hasData\":[true],\"time\":\"1538265600\",\"value\":[82]},{\"formattedAxisTime\":\"Oct 7, 2018\",\"formattedTime\":\"Oct 7 - Oct 13 2018\",\"formattedValue\":[\"89\"],\"hasData\":[true],\"isPartial\":true,\"time\":\"1538870400\",\"value\":[89]}]}}"
	//js, err := simplejson.NewJson([]byte(data))
	//if err != nil {
	//	logs.Error("get_data json err:%v", err)
	//	return nil
	//}
	if js == nil {
		return nil
	}
	timeline := js.Get("default").Get("timelineData")
	arr, err := timeline.Array()
	if err != nil {
		logs.Error("timeline.Array() err:%s", err)
		return nil
	}
	var ret []*WeekData
	for idx, _ := range arr {
		week := timeline.GetIndex(idx)
		wd := &WeekData{}
		start, _ := week.Get("formattedAxisTime").String()
		start = strings.Replace(strings.ToLower(start), ",", "", -1)
		starts := strings.Split(start, " ")
		wd.StartY, _ = strconv.Atoi(starts[2])
		wd.StartM = Month[starts[0]]
		wd.StartD, _ = strconv.Atoi(starts[1])

		end, _ := week.Get("formattedTime").String()
		end = strings.Replace(strings.ToLower(end), " - ", " ", -1)
		ends := strings.Split(end, " ")
		if len(ends) == 5 {
			ends = ends[2:]
		} else {
			ends = ends[3:]
		}

		wd.EndY, _ = strconv.Atoi(ends[2])
		wd.EndM = Month[ends[0]]
		wd.EndD, _ = strconv.Atoi(ends[1])

		values, _ := week.Get("value").Array()
		if len(values) > 0 {
			wd.Value, _ = strconv.Atoi(string(values[0].(json.Number)))
		}
		_, wd.IsPartial = week.CheckGet("isPartial")
		ret = append(ret, wd)
		//logs.Info("start:%d-%d-%d  end:%d-%d-%d value:%d", startY, startM, startD, endY, endM, endD, value)
	}
	return ret
}
