package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	kBaseUrl                = "https://apptoogoodtogo.com/api/"
	kAuthByEmailEndpoint    = "auth/v3/authByEmail"
	kAuthByRequestPollingId = "auth/v3/authByRequestPollingId"
	kRefreshTokenEndpoint   = "auth/v3/token/refresh"

	kApiListOpenedOrders = "order/v6/active"

	kApiItemEndpoint = "item/v7/"

	kDefaultTokenLifeTime = 4 * time.Hour
)

type TooGooToGoClient struct {
	Config            *TooGoodToGoConfig `json:"-"`
	ApkVersion        string             `json:"apkVersion"`
	AccessToken       string             `json:"accessToken"`
	RefreshToken      string             `json:"refreshToken"`
	Cookie            []string           `json:"cookie"`
	UserId            string             `json:"userId"`
	UserAgent         string             `json:"userAgent"`
	LastRefreshedTime time.Time          `json:"lastRefreshedTime"`

	lastQueryTime             time.Time    `json:"-"`
	lastOpenedOrdersQueryTime time.Time    `json:"-"`
	httpClient                *http.Client `json:"-"`
	captchaSolved             bool         `json:"-"`
	verbose                   bool         `json:"-"`
}

func NewTooGooToGoClient(config *TooGoodToGoConfig, verbose bool) *TooGooToGoClient {
	lastApkVersion, err := GetLastApkVersion()
	if err != nil {
		glog.Fatalf("error from GetLastApkVersion: %v", err)
	}

	randomUserAgentPos := rand.Intn(3)

	var userAgent string
	if randomUserAgentPos == 0 {
		userAgent = fmt.Sprintf("TGTG/%v Dalvik/2.1.0 (Linux; U; Android 9; Nexus 5 Build/M4B30Z)", lastApkVersion)
	} else if randomUserAgentPos == 1 {
		userAgent = fmt.Sprintf("TGTG/%v Dalvik/2.1.0 (Linux; U; Android 10; SM-G935F Build/NRD90M)", lastApkVersion)
	} else {
		userAgent = fmt.Sprintf("TGTG/%v Dalvik/2.1.0 (Linux; Android 12; SM-G920V Build/MMB29K)", lastApkVersion)
	}

	return &TooGooToGoClient{
		Config:     config,
		ApkVersion: lastApkVersion,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		UserAgent: userAgent,
		verbose:   verbose,
	}
}

func (client *TooGooToGoClient) IsLoggedIn() bool {
	return len(client.AccessToken) > 0 && len(client.RefreshToken) > 0 && len(client.UserId) > 0
}

func (client *TooGooToGoClient) IsTokenStillValid() bool {
	return client.LastRefreshedTime.Add(kDefaultTokenLifeTime).After(time.Now())
}

func (client *TooGooToGoClient) refreshToken() error {
	if client.IsTokenStillValid() {
		return nil
	}

	jsonData := fmt.Sprintf(`{"refresh_token": "%v"}`, client.RefreshToken)

	response, err := client.Query("POST", kRefreshTokenEndpoint, []byte(jsonData))
	if err != nil {
		return fmt.Errorf("error from client.Query: %w", err)
	}

	var parsedBody map[string]interface{}
	err = json.Unmarshal([]byte(response.Body), &parsedBody)
	if err != nil {
		return fmt.Errorf("error from json.Unmarshal: %w", err)
	}
	client.AccessToken = parsedBody["access_token"].(string)
	client.RefreshToken = parsedBody["refresh_token"].(string)
	client.LastRefreshedTime = time.Now()

	err = client.writeAuthorizationDataToFile()
	if err != nil {
		glog.Printf("error in client.WriteAuthorizationDataToFile: %v\n", err)
	}

	return nil
}

func (client *TooGooToGoClient) writeAuthorizationDataToFile() error {
	file, err := json.MarshalIndent(client, "", " ")
	if err != nil {
		return fmt.Errorf("error in json.MarshalIndent")
	}

	latestFileName := client.latestAuthorizationFileName()
	err = os.WriteFile(latestFileName, file, 0644)
	if err != nil {
		return fmt.Errorf("error in ioutil.WriteFile")
	}
	glog.Printf("dumped authorization data to %v\n", latestFileName)

	return nil
}

func (client *TooGooToGoClient) latestAuthorizationFileName() string {
	return fmt.Sprintf("secrets/tooGoodToGoClient.%v.latest.json", client.Config.AccountEmail)
}

func (client *TooGooToGoClient) removeLatestAuthorizationFileName() {
	latestFileName := client.latestAuthorizationFileName()

	err := os.Remove(latestFileName)
	if err != nil {
		glog.Printf("error in os.Remove: %v\n", err)
	} else {
		glog.Printf("deleted file %v\n", latestFileName)
	}
}

func (client *TooGooToGoClient) readAuthorizationDataFromLatestFile() error {
	latestFileName := client.latestAuthorizationFileName()

	fileData, err := os.ReadFile(latestFileName)
	if os.IsNotExist(err) {
		return err
	}
	if err != nil {
		defer client.removeLatestAuthorizationFileName()
		return fmt.Errorf("error in os.ReadFile: %w", err)
	}

	err = json.Unmarshal([]byte(fileData), client)
	if err != nil {
		defer client.removeLatestAuthorizationFileName()
		return fmt.Errorf("error in json.Unmarshal: %w", err)
	}

	glog.Printf("read authorization data from %v\n", latestFileName)

	return nil
}

func (client *TooGooToGoClient) LoginOrRefreshToken() error {
	if client.IsLoggedIn() {
		return client.refreshToken()
	}

	err := client.readAuthorizationDataFromLatestFile()
	if os.IsNotExist(err) {
		// file does not exist, no error - just proceed to login
		err = nil
	} else if err != nil {
		return fmt.Errorf("error in readAuthorizationDataFromLatestFile: %w\n", err)
	} else if client.IsTokenStillValid() {
		return nil
	} else {
		client.removeLatestAuthorizationFileName()
	}

	glog.Printf("too good to go log in for %v...\n", client.Config.AccountEmail)

	jsonDataBeg := fmt.Sprintf(`{
		"device_type": "ANDROID",
		"email": "%v"`,
		client.Config.AccountEmail,
	)

	authData := jsonDataBeg + "}"

	response, err := client.Query("POST", kAuthByEmailEndpoint, []byte(authData))
	if err != nil {
		return fmt.Errorf("error from client.Query: %w", err)
	}

	var parsedResponse map[string]string
	err = json.Unmarshal([]byte(response.Body), &parsedResponse)
	if err != nil {
		return fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	state, hasState := parsedResponse["state"]
	if !hasState {
		return fmt.Errorf("unexpected response %v\n", parsedResponse)
	}

	if state == "TERMS" {
		return fmt.Errorf("email %v does not seem to be associated with a too good to go account, retry with another mail", client.Config.AccountEmail)
	}
	if state == "WAIT" {
		pollingId, hasPollingId := parsedResponse["polling_id"]
		if !hasPollingId {
			return fmt.Errorf("expected field 'polling_id' in response %v", parsedResponse)
		}
		jsonDataPolling := jsonDataBeg + fmt.Sprintf(`, "request_polling_id": "%v"}`, pollingId)

		err = client.initiateLogin(jsonDataPolling)
		if err != nil {
			return fmt.Errorf("error from initiateLogin: %w", err)
		}
		return nil
	}

	return fmt.Errorf("unexpected state %v in log in response body %v", state, parsedResponse)
}

func (client *TooGooToGoClient) initiateLogin(jsonDataPolling string) error {
	glog.Printf("too good to go validation email sent to %v - you need to validate login", client.Config.AccountEmail)
	const kMaxPollingRetries = 20

	for retryPos := 0; retryPos < kMaxPollingRetries; retryPos++ {
		glog.Printf("check %v/%v validation (check %v inbox)...\n", retryPos, kMaxPollingRetries, client.Config.AccountEmail)
		response, err := client.Query("POST", kAuthByRequestPollingId, []byte(jsonDataPolling))
		if err != nil {
			return fmt.Errorf("error from client.Query: %w", err)
		}

		if len(response.Body) > 0 {
			var parsedBody map[string]interface{}
			err = json.Unmarshal([]byte(response.Body), &parsedBody)
			if err != nil {
				return fmt.Errorf("error from json.Unmarshal: %w", err)
			}
			client.AccessToken = parsedBody["access_token"].(string)
			client.RefreshToken = parsedBody["refresh_token"].(string)
			client.LastRefreshedTime = time.Now()

			client.UserId = parsedBody["startup_data"].(map[string]interface{})["user"].(map[string]interface{})["user_id"].(string)

			err = client.writeAuthorizationDataToFile()
			if err != nil {
				glog.Printf("error in dumpAuthorizationDataToFile: %v\n", err)
			}

			glog.Printf("logged in successfully!")
			return nil
		}
	}
	return fmt.Errorf("max retries exceeded for polling, retry and accept validation email")
}

type ItemParameters struct {
	UserId        string   `json:"user_id"`
	Origin        Location `json:"origin"`
	Radius        int      `json:"radius"`
	PageSize      int      `json:"page_size"`
	Page          int      `json:"page"`
	Discover      bool     `json:"discover"`
	FavoritesOnly bool     `json:"favorites_only"`
	WithStockOnly bool     `json:"with_stock_only"`
}

func (client *TooGooToGoClient) ListStores() ([]Store, error) {
	err := client.LoginOrRefreshToken()
	if err != nil {
		return []Store{}, fmt.Errorf("error from login: %w", err)
	}

	searchConfig := &client.Config.SearchConfig

	params := ItemParameters{
		UserId:        client.UserId,
		Origin:        searchConfig.Origin,
		Radius:        searchConfig.RadiusInKm,
		PageSize:      20,
		Page:          1,
		Discover:      false,
		FavoritesOnly: searchConfig.FavoritesOnly,
		WithStockOnly: searchConfig.WithStockOnly,
	}

	jsonParams, err := json.Marshal(params)
	if err != nil {
		return []Store{}, fmt.Errorf("error from json.Marshal: %w", err)
	}

	response, err := client.Query("POST", kApiItemEndpoint, jsonParams)
	if err != nil {
		return []Store{}, fmt.Errorf("error from client.Query: %w", err)
	}

	stores, err := CreateStoresFromListStoresResponse(response.Body)
	if err != nil {
		return stores, fmt.Errorf("error from CreateStoresFromListStoresResponse: %w", err)
	}

	if len(stores) > 0 {
		glog.Printf("found %v store(s)!\n", len(stores))
	}

	return stores, err
}

type OpenedOrdersParameters struct {
	UserId string `json:"user_id"`
}

func (client *TooGooToGoClient) ListOpenedOrders() ([]Order, error) {
	if !client.canListOpenedOrders() {
		return []Order{}, nil
	}
	err := client.LoginOrRefreshToken()
	if err != nil {
		return []Order{}, fmt.Errorf("error from login: %w", err)
	}

	params := OpenedOrdersParameters{
		UserId: client.UserId,
	}

	jsonParams, err := json.Marshal(params)
	if err != nil {
		return []Order{}, fmt.Errorf("error from json.Marshal: %w", err)
	}

	response, err := client.Query("POST", kApiListOpenedOrders, jsonParams)
	if err != nil {
		return []Order{}, fmt.Errorf("error from client.Query: %w", err)
	}

	openedOrders, err := CreateOrdersFromListOrdersResponse(response.Body)
	if err != nil {
		return openedOrders, fmt.Errorf("error from CreateStoresFromListStoresResponse: %w", err)
	}

	if len(openedOrders) > 0 {
		glog.Printf("you have %v order(s) to pickup, don't forget them:\n", len(openedOrders))
		for orderPos, openedOrder := range openedOrders {
			glog.Printf("- Order %v - %v\n", orderPos+1, openedOrder)
		}
	}

	return openedOrders, err
}

func (client *TooGooToGoClient) addHeaders(req *http.Request) {
	req.Header.Add("Accept", "application/json")
	if client.Config.UseGzipEncoding {
		req.Header.Add("Accept-Encoding", "gzip")
	}
	req.Header.Add("Accept-Language", client.Config.Language)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	req.Header.Add("User-Agent", client.UserAgent)
	for _, cookieVal := range client.Cookie {
		req.Header.Add("Cookie", cookieVal)
	}
	if len(client.AccessToken) > 0 {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", client.AccessToken))
	}
}

type QueryResponse struct {
	Body       string
	StatusCode int
}

func printHeaders(url *url.URL, title string, header *http.Header) {
	glog.Printf("  %v %v headers:\n", url, title)
	for headerName, headerValue := range *header {
		glog.Printf("  - %v: %v\n", headerName, strings.Join(headerValue, "; "))
	}
}

func hasGzipContentEncodingHeader(header *http.Header) bool {
	contentEncoding, hasContentEncoding := (*header)["Content-Encoding"]
	if hasContentEncoding {
		for _, contentEncodingPart := range contentEncoding {
			if contentEncodingPart == "gzip" {
				return true
			}
		}
	}
	return false
}

func (client *TooGooToGoClient) Query(method, path string, body []byte) (QueryResponse, error) {
	url, err := url.JoinPath(kBaseUrl, path)
	var ret QueryResponse
	if err != nil {
		return ret, fmt.Errorf("error from url.JoinPath: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return ret, fmt.Errorf("error from http.NewRequest: %w", err)
	}

	client.addHeaders(req)

	client.sleepIfNeeded()

	if client.verbose && len(req.Header) > 0 {
		printHeaders(req.URL, "request", &req.Header)
	}

	glog.Printf("calling %v\n", req.URL)
	res, err := client.httpClient.Do(req)
	if err != nil {
		return ret, fmt.Errorf("error from client.Client.Do: %w", err)
	}
	defer res.Body.Close()

	if client.verbose {
		printHeaders(req.URL, "response", &res.Header)
	}

	ret.StatusCode = res.StatusCode

	var resBodyReader io.Reader

	if hasGzipContentEncodingHeader(&res.Header) {
		gzReader, err := gzip.NewReader(res.Body)
		if err != nil {
			return ret, fmt.Errorf("error from gzip.NewReader: %w", err)
		}
		defer gzReader.Close()

		resBodyReader = gzReader
	} else {
		resBodyReader = res.Body
	}

	uncompressedResponse, err := io.ReadAll(resBodyReader)
	if err != nil {
		return ret, fmt.Errorf("error from io.ReadAll: %w", err)
	}

	retry, err := client.checkCaptcha(uncompressedResponse)
	if retry {
		return client.Query(method, path, body)
	}

	ret.Body = string(uncompressedResponse)

	client.setCookie(&res.Header)

	return ret, nil
}

func (client *TooGooToGoClient) setCookie(header *http.Header) {
	cookies, hasSetCookie := (*header)["Set-Cookie"]
	if hasSetCookie {
		client.Cookie = cookies
	}
}

func (client *TooGooToGoClient) checkCaptcha(uncompressedResponse []byte) (bool, error) {
	var parsedResponse map[string]string
	err := json.Unmarshal(uncompressedResponse, &parsedResponse)
	if err != nil {
		return false, nil
	}

	urlCaptcha, hasUrlCaptcha := parsedResponse["url"]
	if hasUrlCaptcha {
		if client.captchaSolved {
			return false, errors.New("new captcha detected - unable to proceed further")
		}
		if strings.HasPrefix(urlCaptcha, "https://geo.captcha-delivery.com") {
			glog.Printf("you need to solve captcha manually")
			err = OpenBrowser(urlCaptcha)
			if err != nil {
				return false, fmt.Errorf("error in OpenBrowser: %w", err)
			}
			client.captchaSolved = true
			return true, nil
		}
	}
	client.captchaSolved = false

	return false, nil
}

func (client *TooGooToGoClient) nextQueryDelay() time.Duration {
	minRequestsPeriod := client.Config.MinRequestsPeriod.Duration
	randomExtraDuration := time.Duration(rand.Int63n(minRequestsPeriod.Nanoseconds()))
	return minRequestsPeriod + randomExtraDuration
}

func (client *TooGooToGoClient) sleepIfNeeded() {
	nowTime := time.Now()
	if !client.lastQueryTime.IsZero() {
		elapsedTimeSinceLastQuery := nowTime.Sub(client.lastQueryTime)
		waitingTime := client.nextQueryDelay() - elapsedTimeSinceLastQuery
		if waitingTime > 0 {
			time.Sleep(waitingTime)
		}
	}
	client.lastQueryTime = nowTime
}

func (client *TooGooToGoClient) canListOpenedOrders() bool {
	nowTime := time.Now()
	if client.lastOpenedOrdersQueryTime.IsZero() {
		client.lastOpenedOrdersQueryTime = nowTime
		return true
	}
	if client.lastOpenedOrdersQueryTime.Add(client.Config.ActiveOrdersReminderPeriod.Duration).Before(nowTime) {
		client.lastOpenedOrdersQueryTime = nowTime
		return true
	}
	return false
}
