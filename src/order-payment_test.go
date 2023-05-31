package tga

import (
	"os"
	"testing"
)

const (
	kExampleOrderPaymentFilepath = "testdata/example_order_payment.json"
)

func TestOrderPaymentFromResponse(t *testing.T) {
	responseBody, err := os.ReadFile(kExampleOrderPaymentFilepath)
	if err != nil {
		t.Fatalf("error reading file %v", kExampleOrderPaymentFilepath)
	}
	orderPayment, err := NewOrderPaymentFromPayOrderResponse(responseBody)
	if err != nil {
		t.Fatalf("error in NewPaymentMethodsFromPaymentMethodsResponse: %v", err)
	}

	expectedOrderPayment := OrderPayment{
		Id:              "123456789",
		OrderId:         "orderid12354",
		PaymentProvider: Adyen,
		State:           "AUTHORIZATION_INITIATED",
	}

	if expectedOrderPayment != orderPayment {
		t.Fatalf("expected %v == %v\n", expectedOrderPayment, orderPayment)
	}

}
