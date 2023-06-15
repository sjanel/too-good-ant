package tga

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	Config                 *TooGoodToGoConfig `json:"-"`
	AccessToken            string             `json:"accessToken"`
	RefreshToken           string             `json:"refreshToken"`
	Cookie                 []string           `json:"cookie"`
	UserId                 string             `json:"userId"`
	UserAgent              string             `json:"userAgent"`
	LastLogInRefreshedTime time.Time          `json:"lastLogInRefreshedTime"`
	LastTokenRefreshedTime time.Time          `json:"lastTokenRefreshedTime"`

	currentAccountPos         int          `json:"-"`
	lastQueryTimePerAccount   []time.Time  `json:"-"`
	lastOpenedOrdersQueryTime time.Time    `json:"-"`
	httpClient                *http.Client `json:"-"`
	verbose                   bool         `json:"-"`
}

func (client TooGooToGoClient) emailAccount() string {
	return client.Config.Accounts[client.currentAccountPos].Email
}

func (client *TooGooToGoClient) resetAuthData() {
	client.AccessToken = ""
	client.RefreshToken = ""
	client.Cookie = []string{}
	client.UserId = ""
}

func NewHttpClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
	}
}

func (client *TooGooToGoClient) incrCurrentAccountPos() {
	nbAccounts := len(client.Config.Accounts)
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

	client.resetAuthData()
	client.httpClient = NewHttpClient()
	client.incrCurrentAccountPos()

	var err error
	client.UserAgent, err = getUserAgent(client.Config, client.currentAccountPos)
	if err != nil {
		glog.Fatalf("error from getUserAgent: %v", err)
	}

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

	err = client.ensureAuthDataValidity()
	if err != nil {
		return fmt.Errorf("error from client.ensureAuthDataValidity: %w", err)
	}
	return nil
}

func getUserAgent(config *TooGoodToGoConfig, accountPos int) (string, error) {
	userAgent := config.Accounts[accountPos].UserAgent
	if len(userAgent) > 0 {
		return userAgent, nil
	}

	lastApkVersion, err := GetLastApkVersion()
	if err != nil {
		return "", fmt.Errorf("error from GetLastApkVersion: %w", err)
	}

	const kDalvikVersion = "2.1.0"

	kUserAgents := [...]string{
		fmt.Sprintf("TGTG/%v Dalvik/%v (Linux; Android 12; SM-G973F Build/SP1A.210812.016; wv)", lastApkVersion, kDalvikVersion),
		fmt.Sprintf("TGTG/%v Dalvik/%v (Linux; Android 12; SM-G975U1 Build/SP1A.210812.016; wv)", lastApkVersion, kDalvikVersion),
		fmt.Sprintf("TGTG/%v Dalvik/%v (Linux; Android 13; SAMSUNG SM-G991U1)", lastApkVersion, kDalvikVersion),
	}

	return kUserAgents[rand.Intn(len(kUserAgents))], nil
}

func NewTooGooToGoClient(config *TooGoodToGoConfig, verbose bool) *TooGooToGoClient {
	firstUserAgent, err := getUserAgent(config, 0)
	if err != nil {
		glog.Fatalf("error from getUserAgent: %v", err)
	}

	lastQueryTimePerAccount := make([]time.Time, len(config.Accounts))

	return &TooGooToGoClient{
		Config:     config,
		httpClient: NewHttpClient(),
		UserAgent:  firstUserAgent,
		verbose:    verbose,

		lastQueryTimePerAccount: lastQueryTimePerAccount,
	}
}

func (client *TooGooToGoClient) IsLoggedIn() bool {
	return len(client.AccessToken) > 0 && len(client.RefreshToken) > 0 && len(client.UserId) > 0
}

func (client *TooGooToGoClient) IsLogInStillValid() bool {
	return client.LastLogInRefreshedTime.Add(client.Config.LogInValidityDuration.Duration).After(time.Now())
}

func (client *TooGooToGoClient) IsTokenStillValid() bool {
	return client.LastTokenRefreshedTime.Add(client.Config.TokenValidityDuration.Duration).After(time.Now())
}

func (client *TooGooToGoClient) setRefreshedTokenData(responseBody []byte) error {
	var parsedBody map[string]interface{}
	err := json.Unmarshal(responseBody, &parsedBody)
	if err != nil {
		return fmt.Errorf("error from json.Unmarshal: %w", err)
	}
	client.AccessToken = parsedBody["access_token"].(string)
	client.RefreshToken = parsedBody["refresh_token"].(string)
	client.LastTokenRefreshedTime = time.Now()

	glog.Printf("refreshed token\n")

	return nil
}

func (client *TooGooToGoClient) refreshToken() error {
	if client.IsTokenStillValid() {
		return nil
	}

	jsonData := fmt.Sprintf(`{"refresh_token": "%v"}`, client.RefreshToken)

	response, err := client.query("POST", kRefreshTokenEndpoint, []byte(jsonData), true)
	if err != nil {
		return fmt.Errorf("error from client.query: %w", err)
	}

	err = client.setRefreshedTokenData(response.Body)
	if err != nil {
		return fmt.Errorf("error in client.setRefreshedTokenData: %w\n", err)
	}

	err = client.writeAuthorizationDataToFile()
	if err != nil {
		glog.Printf("error in client.writeAuthorizationDataToFile: %v\n", err)
	}

	return nil
}

func (client *TooGooToGoClient) writeAuthorizationDataToFile() error {
	file, err := json.MarshalIndent(client, "", " ")
	if err != nil {
		return fmt.Errorf("error in json.MarshalIndent: %w", err)
	}

	latestFileName := client.latestAuthorizationFileName()
	err = os.WriteFile(latestFileName, file, 0644)
	if err != nil {
		return fmt.Errorf("error in ioutil.WriteFile: %w", err)
	}
	glog.Printf("wrote authorization data to %v\n", latestFileName)

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

	err = json.Unmarshal(fileData, client)
	if err != nil {
		defer client.removeLatestAuthorizationFileName()
		return fmt.Errorf("error in json.Unmarshal: %w", err)
	}

	glog.Printf("read authorization data from %v\n", latestFileName)

	return nil
}

func (client *TooGooToGoClient) logIn() error {
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
	err = json.Unmarshal(response.Body, &parsedResponse)
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
	if state != "WAIT" {
		return fmt.Errorf("unexpected state %v in log in response body %v", state, parsedResponse)
	}

	pollingId, hasPollingId := parsedResponse["polling_id"]
	if !hasPollingId {
		return fmt.Errorf("expected field 'polling_id' in response %v", parsedResponse)
	}
	jsonDataPolling := jsonDataBeg + fmt.Sprintf(`, "request_polling_id": "%v"}`, pollingId)

	err = client.initiateLogin(jsonDataPolling)
	if err != nil {
		return fmt.Errorf("error from initiateLogin: %w", err)
	}

	client.LastLogInRefreshedTime = time.Now()

	glog.Printf("logged in successfully\n")

	return nil
}

func (client *TooGooToGoClient) ensureAuthDataValidity() error {
	if client.IsLoggedIn() {
		return client.refreshToken()
	}

	err := client.readAuthorizationDataFromLatestFile()
	if os.IsNotExist(err) {
		// file does not exist, no error - just proceed to login
		err = nil
	} else if err != nil {
		return fmt.Errorf("error in readAuthorizationDataFromLatestFile: %w\n", err)
	} else if client.IsLogInStillValid() {
		return nil
	} else {
		glog.Printf("authorization data has expired\n")
		client.removeLatestAuthorizationFileName()
		client.resetAuthData()
	}

	err = client.logIn()
	if err != nil {
		return fmt.Errorf("error from client.logIn: %w", err)
	}
	return nil
}

func (client *TooGooToGoClient) setUserId() error {
	// Should be logged in
	response, err := client.query("POST", kApiUserInformation, []byte{}, false)
	if err != nil {
		return fmt.Errorf("error from client.query: %w", err)
	}

	var parsedBody map[string]interface{}
	err = json.Unmarshal(response.Body, &parsedBody)
	if err != nil {
		return fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	client.UserId = parsedBody["user_id"].(string)

	return nil
}

func (client *TooGooToGoClient) initiateLogin(jsonDataPolling string) error {
	initiateLoginTime := time.Now()
	timeoutTime := initiateLoginTime.Add(client.Config.LogInEmailValidationTimeoutDuration.Duration)

	glog.Printf("check %v inbox and validate log in in email link before %v\n", client.emailAccount(), timeoutTime)

	for timeoutTime.After(time.Now()) {
		response, err := client.query("POST", kAuthByRequestPollingId, []byte(jsonDataPolling), true)
		if err != nil {
			return fmt.Errorf("error from client.Query: %w", err)
		}

		if len(response.Body) > 0 {
			err = client.setRefreshedTokenData(response.Body)
			if err != nil {
				return fmt.Errorf("error from client.setRefreshedTokenData: %w", err)
			}

			err = client.setUserId()
			if err != nil {
				return fmt.Errorf("error from client.setUserId: %w", err)
			}

			err = client.writeAuthorizationDataToFile()
			if err != nil {
				glog.Printf("error in client.writeAuthorizationDataToFile: %v\n", err)
			}

			return nil
		}
	}
	return fmt.Errorf("authentication validation timeout")
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
		glog.Printf("found %v store(s)\n", len(stores))
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
	err := client.ensureAuthDataValidity()
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
	req.Header.Add("Accept-Encoding", "gzip")
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
	Body       []byte
	StatusCode int
}

func printHeaders(url *url.URL, title string, header *http.Header) {
	glog.Printf("  %v %v headers:\n", url, title)
	for headerName, headerValue := range *header {
		glog.Printf("  - %v: %v\n", headerName, strings.Join(headerValue, "; "))
	}
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

	ret.Body, err = DecompressAllBody(res)
	if err != nil {
		return ret, fmt.Errorf("error from DecompressAllBody: %w", err)
	}

	retry, err = client.checkCaptcha(ret.Body)
	if retry {
		return client.query(method, path, body, sleepIfNeeded)
	}

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
		err := client.logIn()
		if err != nil {
			return false, fmt.Errorf("error from client.logIn: %w", err)
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

func (client *TooGooToGoClient) Close() error {
	client.httpClient.CloseIdleConnections()

	err := client.writeAuthorizationDataToFile()
	if err != nil {
		return fmt.Errorf("error in client.writeAuthorizationDataToFile: %w\n", err)
	}
	return nil
}
