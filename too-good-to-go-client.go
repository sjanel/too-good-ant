package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
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
	kAuthByEmailEndpoint    = "auth/v4/authByEmail"
	kAuthByRequestPollingId = "auth/v4/authByRequestPollingId"
	kRefreshTokenEndpoint   = "auth/v3/token/refresh"

	kApiListOpenedOrders = "order/v7/active"
	kApiCreateOrder      = "order/v7/create"

	kApiUserInformation = "user/v2"

	kApiPaymentMethods = "paymentMethod/v1/"

	kApiItemEndpoint = "item/v7/"
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

	currentAccountPos         int          `json:"-"`
	lastQueryTimePerAccount   []time.Time  `json:"-"`
	lastOpenedOrdersQueryTime time.Time    `json:"-"`
	httpClient                *http.Client `json:"-"`
	verbose                   bool         `json:"-"`
}

func (client TooGooToGoClient) emailAccount() string {
	return client.Config.AccountsEmail[client.currentAccountPos]
}

func NewHttpClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
	}
}

func (client *TooGooToGoClient) incrCurrentAccountPos() {
	nbAccounts := len(client.Config.AccountsEmail)
	if nbAccounts == 1 {
		return
	}

	client.currentAccountPos++
	if client.currentAccountPos == nbAccounts {
		client.currentAccountPos = 0
	}

	glog.Printf("switched to too good to go account %v\n", client.emailAccount())
}

func (client *TooGooToGoClient) switchToNextEmailAccount() error {

	client.AccessToken = ""
	client.httpClient = NewHttpClient()
	client.Cookie = []string{}

	client.incrCurrentAccountPos()

	if client.currentAccountPos == 0 {
		tooManyRequestsPauseDuration := client.Config.TooManyRequestsPausePeriod.Duration
		minTimeBeforeNextRequest := client.lastQueryTime().Add(tooManyRequestsPauseDuration)
		nowTime := time.Now()
		if nowTime.Before(minTimeBeforeNextRequest) {
			waitingDuration := minTimeBeforeNextRequest.Sub(nowTime)
			glog.Printf("waiting %v as too many requests reached\n", waitingDuration)
			time.Sleep(waitingDuration)
		}
	}

	err := client.loginOrRefreshToken()
	if err != nil {
		return fmt.Errorf("error from client.LoginOrRefreshToken: %w", err)
	}
	return nil
}

func NewTooGooToGoClient(config *TooGoodToGoConfig, verbose bool) *TooGooToGoClient {
	lastApkVersion, err := GetLastApkVersion()
	if err != nil {
		glog.Fatalf("error from GetLastApkVersion: %v", err)
	}

	kUserAgents := [...]string{
		fmt.Sprintf("TGTG/%v Dalvik/2.1.0 (Linux; U; Android 9; Nexus 5 Build/M4B30Z)", lastApkVersion),
		fmt.Sprintf("TGTG/%v Dalvik/2.1.0 (Linux; U; Android 10; SM-G935F Build/NRD90M)", lastApkVersion),
		fmt.Sprintf("TGTG/%v Dalvik/2.1.0 (Linux; Android 12; SM-G920V Build/MMB29K)", lastApkVersion),
	}

	userAgent := kUserAgents[rand.Intn(len(kUserAgents))]
	lastQueryTimePerAccount := make([]time.Time, len(config.AccountsEmail))

	return &TooGooToGoClient{
		Config:     config,
		ApkVersion: lastApkVersion,
		httpClient: NewHttpClient(),
		UserAgent:  userAgent,
		verbose:    verbose,

		lastQueryTimePerAccount: lastQueryTimePerAccount,
	}
}

func (client *TooGooToGoClient) IsLoggedIn() bool {
	return len(client.AccessToken) > 0 && len(client.RefreshToken) > 0 && len(client.UserId) > 0
}

func (client *TooGooToGoClient) IsTokenStillValid() bool {
	return client.LastRefreshedTime.Add(client.Config.TokenValidityDuration.Duration).After(time.Now())
}

func (client *TooGooToGoClient) refreshToken() error {
	if client.IsTokenStillValid() {
		return nil
	}

	jsonData := fmt.Sprintf(`{"refresh_token": "%v"}`, client.RefreshToken)

	response, err := client.query("POST", kRefreshTokenEndpoint, []byte(jsonData), true)
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
	return fmt.Sprintf("secrets/tooGoodToGoClient.%v.latest.json", client.emailAccount())
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

func (client *TooGooToGoClient) loginOrRefreshToken() error {
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
		glog.Printf("authorization data has expired\n")
		client.removeLatestAuthorizationFileName()
	}

	glog.Printf("too good to go log in for %v...\n", client.emailAccount())

	jsonDataBeg := fmt.Sprintf(`{
		"device_type": "ANDROID",
		"email": "%v"`,
		client.emailAccount(),
	)

	authData := jsonDataBeg + "}"

	response, err := client.query("POST", kAuthByEmailEndpoint, []byte(authData), true)
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
		return fmt.Errorf("email %v does not seem to be associated with a too good to go account, retry with another mail", client.emailAccount())
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
	glog.Printf("too good to go validation email sent to %v - you need to validate login", client.emailAccount())
	const kMaxPollingRetries = 20

	for retryPos := 0; retryPos < kMaxPollingRetries; retryPos++ {
		glog.Printf("check %v/%v validation (check %v inbox)...\n", retryPos+1, kMaxPollingRetries, client.emailAccount())
		response, err := client.query("POST", kAuthByRequestPollingId, []byte(jsonDataPolling), true)
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

			response, err := client.query("POST", kApiUserInformation, []byte{}, false)
			if err != nil {
				return fmt.Errorf("error from client.Query: %w", err)
			}

			err = json.Unmarshal([]byte(response.Body), &parsedBody)
			if err != nil {
				return fmt.Errorf("error from json.Unmarshal: %w", err)
			}

			client.UserId = parsedBody["user_id"].(string)

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

	response, err := client.postQueryWithSleep(kApiItemEndpoint, params)
	if err != nil {
		return []Store{}, fmt.Errorf("error from client.postQueryWithSleep: %w", err)
	}

	stores, err := NewStoresFromListStoresResponse(response.Body)
	if err != nil {
		return stores, fmt.Errorf("error from NewStoresFromListStoresResponse: %w", err)
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

	params := OpenedOrdersParameters{
		UserId: client.UserId,
	}

	response, err := client.postQueryWithSleep(kApiListOpenedOrders, params)
	if err != nil {
		return []Order{}, fmt.Errorf("error from client.postQueryWithSleep: %w", err)
	}

	openedOrders, err := NewOrdersFromListOrdersResponse(response.Body)
	if err != nil {
		return openedOrders, fmt.Errorf("error from NewOrdersFromListOrdersResponse: %w", err)
	}

	if len(openedOrders) > 0 {
		glog.Printf("you have %v order(s) to pickup, don't forget them:\n", len(openedOrders))
		for orderPos, openedOrder := range openedOrders {
			glog.Printf("- Order %v - %v\n", orderPos+1, openedOrder)
		}
	}

	return openedOrders, err
}

type PaymentMethodsParameters struct {
	PaymentMethodRequestItem []PaymentMethodRequestItem `json:"supported_types"`
}

type PaymentMethodRequestItem struct {
	PaymentProvider PaymentProvider `json:"provider"`
	PaymentTypes    []PaymentType   `json:"payment_types"`
}

func (client *TooGooToGoClient) PaymentMethods(paymentProvider PaymentProvider) ([]PaymentMethod, error) {
	params := PaymentMethodsParameters{
		PaymentMethodRequestItem: []PaymentMethodRequestItem{
			{
				PaymentProvider: paymentProvider,
				PaymentTypes: []PaymentType{
					CreditCard,
					PayPal,
					GooglePay,
				},
			},
		},
	}

	response, err := client.postQueryWithSleep(kApiPaymentMethods, params)
	if err != nil {
		return []PaymentMethod{}, fmt.Errorf("error from client.postQueryWithSleep: %w", err)
	}

	paymentMethods, err := NewPaymentMethodsFromPaymentMethodsResponse(response.Body)
	if err != nil {
		return paymentMethods, fmt.Errorf("error from NewPaymentMethodsFromPaymentMethodsResponse: %w", err)
	}

	if len(paymentMethods) > 0 {
		glog.Printf("you have %v payment methods\n", len(paymentMethods))
		for paymentMethodPos, paymentMethod := range paymentMethods {
			glog.Printf("- Payment method %v: %v\n", paymentMethodPos+1, paymentMethod)
		}
	}

	return paymentMethods, nil
}

type CreateOrderParameters struct {
	NbBags int `json:"item_count"`
}

func (client *TooGooToGoClient) ReserveOrder(store Store, nbBags int) (ReservedOrder, error) {
	var reservedOrder ReservedOrder
	if store.AvailableBags < nbBags {
		return reservedOrder, fmt.Errorf("not enough available bags for %v", store)
	}
	params := CreateOrderParameters{
		NbBags: nbBags,
	}

	path := fmt.Sprintf("%v/%v", kApiCreateOrder, store.Id)

	response, err := client.postQueryWithoutSleep(path, params)
	if err != nil {
		return reservedOrder, fmt.Errorf("error from client.postQueryWithoutSleep: %w", err)
	}

	reservedOrder, err = NewReservedOrderFromCreateOrder(response.Body)
	if err != nil {
		return reservedOrder, fmt.Errorf("error from NewReservedOrderFromCreateOrder: %w", err)
	}

	return reservedOrder, nil
}

type CancelOrderParameters struct {
	CancelReason int `json:"cancel_reason_id"`
}

func (client *TooGooToGoClient) CancelOrder(orderId string) error {
	params := CancelOrderParameters{
		CancelReason: 1,
	}

	path := fmt.Sprintf("order/v7/%v/abort", orderId)

	response, err := client.postQueryWithSleep(path, params)
	if err != nil {
		return fmt.Errorf("error from client.postQueryWithoutSleep: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("http error %v, with body: %v", response.StatusCode, response.Body)
	}

	return nil
}

type PayOrderParameters struct {
	Authorization Authorization `json:"authorization"`
}

type Authorization struct {
	AuthorizationPayload AuthorizationPayload `json:"authorization_payload"`
	PaymentProvider      PaymentProvider      `json:"payment_provider"`
	ReturnUrl            string               `json:"return_url"`
}

type AuthorizationPayload struct {
	Type              string      `json:"type"`
	PaymentType       PaymentType `json:"payment_type"`
	Payload           string      `json:"payload"`
	DetailsPayload    string      `json:"details_payload,omitempty"`
	SavePaymentMethod string      `json:"save_payment_method,omitempty"`
}

func (client *TooGooToGoClient) PayOrder(orderId string, paymentMethod PaymentMethod) (OrderPayment, error) {
	params := PayOrderParameters{
		Authorization: Authorization{
			AuthorizationPayload: AuthorizationPayload{
				Type:              paymentMethod.PaymentProvider.GetAuthorizationPayloadType(),
				PaymentType:       paymentMethod.PaymentType,
				Payload:           paymentMethod.AdyenApiPayload, // may not work with non Adyen payment types...
				SavePaymentMethod: paymentMethod.SavePaymentMethod,
			},
			PaymentProvider: paymentMethod.PaymentProvider,
			ReturnUrl:       "adyencheckout://com.app.tgtg.itemview", // TODO: not yet figured out how to set this field
		},
	}

	path := fmt.Sprintf("order/v7/%v/pay", orderId)

	var orderPayment OrderPayment

	response, err := client.postQueryWithSleep(path, params)
	if err != nil {
		return orderPayment, fmt.Errorf("error from client.postQueryWithoutSleep: %w", err)
	}

	orderPayment, err = NewOrderPaymentFromPayOrderResponse(response.Body)
	if err != nil {
		return orderPayment, fmt.Errorf("error from NewOrderPaymentFromPayOrderResponse: %w", err)
	}

	glog.Printf("order payment %v created\n", orderPayment)

	paymentInfoResponse, err := client.postQueryWithSleep(fmt.Sprintf("payment/v3/%v", orderPayment.Id), []byte{})
	if err != nil {
		glog.Printf("error from client.postQueryWithSleep: %v\n", err)
		err = nil
	}

	glog.Printf("payment information from order payment is %v\n", paymentInfoResponse)

	return orderPayment, nil
}

func (client *TooGooToGoClient) postQueryWithSleep(path string, paramObject any) (QueryResponse, error) {
	return client.postQuery(path, paramObject, true)
}

func (client *TooGooToGoClient) postQueryWithoutSleep(path string, paramObject any) (QueryResponse, error) {
	return client.postQuery(path, paramObject, false)
}

func (client *TooGooToGoClient) postQuery(path string, paramObject any, sleepIfNeeded bool) (QueryResponse, error) {
	var ret QueryResponse
	err := client.loginOrRefreshToken()
	if err != nil {
		return ret, fmt.Errorf("error from client.LoginOrRefreshToken: %w", err)
	}

	jsonParams, err := json.Marshal(paramObject)
	if err != nil {
		return ret, fmt.Errorf("error from json.Marshal: %w", err)
	}

	ret, err = client.query("POST", path, jsonParams, sleepIfNeeded)
	if err != nil {
		return ret, fmt.Errorf("error from client.Query: %w", err)
	}

	return ret, nil
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

func (client *TooGooToGoClient) query(method, path string, body []byte, sleepIfNeeded bool) (QueryResponse, error) {
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

	if sleepIfNeeded {
		client.sleepIfNeeded()
	}

	if client.verbose && len(req.Header) > 0 {
		printHeaders(req.URL, "request", &req.Header)
	}

	glog.Printf("%v %v\n", req.Method, req.URL)
	res, err := client.httpClient.Do(req)
	if err != nil {
		return ret, fmt.Errorf("error from client.Client.Do: %w", err)
	}
	defer res.Body.Close()

	if client.verbose {
		printHeaders(req.URL, "response", &res.Header)
	}

	retry, err := client.checkStatusCode(res.StatusCode)
	if retry {
		return client.query(method, path, body, sleepIfNeeded)
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

	retry, err = client.checkCaptcha(uncompressedResponse)
	if retry {
		return client.query(method, path, body, sleepIfNeeded)
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

func (client *TooGooToGoClient) checkStatusCode(statusCode int) (bool, error) {
	switch statusCode {
	case http.StatusOK:
		return false, nil
	case http.StatusUnauthorized:
		glog.Printf("http status %v received, login again\n", statusCode)
		client.removeLatestAuthorizationFileName()
		// force re-login
		*client = *NewTooGooToGoClient(client.Config, client.verbose)

		err := client.loginOrRefreshToken()
		if err != nil {
			return false, fmt.Errorf("error from client.LoginOrRefreshToken: %w", err)
		}
		return true, nil
	default:
		return false, fmt.Errorf("http status %v received\n", statusCode)
	}
}

func (client *TooGooToGoClient) checkCaptcha(uncompressedResponse []byte) (bool, error) {
	var parsedResponse map[string]string
	err := json.Unmarshal(uncompressedResponse, &parsedResponse)
	if err != nil {
		return false, nil
	}

	urlCaptcha, hasUrlCaptcha := parsedResponse["url"]
	if hasUrlCaptcha && strings.HasPrefix(urlCaptcha, "https://geo.captcha-delivery.com") {
		glog.Printf("captcha detected\n")
		err = OpenBrowser(urlCaptcha)
		if err != nil {
			return false, fmt.Errorf("error from OpenBrowser: %w", err)
		}
		err = client.switchToNextEmailAccount()
		if err != nil {
			return false, fmt.Errorf("error from client.switchToNextEmailAccount: %w\n", err)
		}
		return true, nil
	}

	return false, nil
}

func (client *TooGooToGoClient) nextQueryDelay() time.Duration {
	minRequestsPeriod := (2 * client.Config.AverageRequestsPeriod.Duration) / 3
	randomExtraDuration := time.Duration(rand.Int63n(minRequestsPeriod.Nanoseconds()))
	return minRequestsPeriod + randomExtraDuration
}

func (client *TooGooToGoClient) lastQueryTime() *time.Time {
	return &client.lastQueryTimePerAccount[client.currentAccountPos]
}

func (client *TooGooToGoClient) sleepIfNeeded() {
	nowTime := time.Now()
	lastQueryTime := client.lastQueryTime()
	if !lastQueryTime.IsZero() {
		elapsedTimeSinceLastQuery := nowTime.Sub(*lastQueryTime)
		waitingTime := client.nextQueryDelay() - elapsedTimeSinceLastQuery
		if waitingTime > 0 {
			time.Sleep(waitingTime)
			nowTime = nowTime.Add(waitingTime)
		}
	}
	*lastQueryTime = nowTime
}

func (client *TooGooToGoClient) canListOpenedOrders() bool {
	nowTime := time.Now()
	if client.lastOpenedOrdersQueryTime.Add(client.Config.ActiveOrdersReminderPeriod.Duration).Before(nowTime) {
		client.lastOpenedOrdersQueryTime = nowTime
		return true
	}
	return false
}
