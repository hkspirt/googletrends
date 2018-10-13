# google trends
API for Google Trends

# 翻译自python版本
[pytrends](github.com/GeneralMills/pytrends)

# Example
```go
proxy := "http://127.0.0.1:14237"
req := googletrends.NewTrendReq("en-US", 360, "US", "today 12-m", proxy, []string{"world"})
resp := req.InterestOverTime()
data := googletrends.ParseInterestOverTime(resp)
logs.Info("get data:%v", data)
```
