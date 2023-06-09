package tga

import (
	"encoding/json"
	"fmt"
)

type OrderPayment struct {
	Id              string
	OrderId         string
	PaymentProvider PaymentProvider
	State           string
}

func NewOrderPaymentFromPayOrderResponse(responseBody []byte) (OrderPayment, error) {
	var parsedOrderPayment map[string]string
	var orderPayment OrderPayment
	err := json.Unmarshal(responseBody, &parsedOrderPayment)
	if err != nil {
		glog.Printf("full response: %v\n", string(responseBody))
		return orderPayment, fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	orderPayment.Id = parsedOrderPayment["payment_id"]
	orderPayment.OrderId = parsedOrderPayment["order_id"]

	orderPayment.PaymentProvider, err = NewPaymentProvider(parsedOrderPayment["payment_provider"])
	if err != nil {
		return orderPayment, fmt.Errorf("error from NewPaymentProvider: %w", err)
	}

	orderPayment.State = parsedOrderPayment["state"]

	return orderPayment, nil
}
