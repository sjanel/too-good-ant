package main

import (
	"os"
	"testing"
)

const (
	kExamplePaymentMethod1 = "data/test/payment-methods/example_payment_methods1.json"
	kExamplePaymentMethod2 = "data/test/payment-methods/example_payment_methods2.json"
	kExamplePaymentMethod3 = "data/test/payment-methods/example_payment_methods3.json"
)

func TestPaymentMethodEmpty1(t *testing.T) {
	responseBody, err := os.ReadFile(kExamplePaymentMethod1)
	if err != nil {
		t.Fatalf("error reading file %v", kExamplePaymentMethod1)
	}
	paymentMethods, err := NewPaymentMethodsFromPaymentMethodsResponse(responseBody)
	if err != nil {
		t.Fatalf("error in NewPaymentMethodsFromPaymentMethodsResponse: %v", err)
	}

	if len(paymentMethods) != 0 {
		t.Fatalf("expected empty payment method list")
	}
}

func TestPaymentMethodEmpty2(t *testing.T) {
	responseBody, err := os.ReadFile(kExamplePaymentMethod2)
	if err != nil {
		t.Fatalf("error reading file %v", kExamplePaymentMethod2)
	}
	paymentMethods, err := NewPaymentMethodsFromPaymentMethodsResponse(responseBody)
	if err != nil {
		t.Fatalf("error in NewPaymentMethodsFromPaymentMethodsResponse: %v", err)
	}

	if len(paymentMethods) != 0 {
		t.Fatalf("expected empty payment method list")
	}
}

func TestPaymentMethodNonEmpty(t *testing.T) {
	responseBody, err := os.ReadFile(kExamplePaymentMethod3)
	if err != nil {
		t.Fatalf("error reading file %v", kExamplePaymentMethod3)
	}
	paymentMethods, err := NewPaymentMethodsFromPaymentMethodsResponse(responseBody)
	if err != nil {
		t.Fatalf("error in NewPaymentMethodsFromPaymentMethodsResponse: %v", err)
	}

	if len(paymentMethods) != 4 {
		t.Fatalf("expected 4 payment methods")
	}

	expectedPaymentMethod1 := PaymentMethod{
		Id:              "BHKI4N2GMWVU2S56",
		InternalType:    "adyenSavedPaymentMethod",
		AdyenApiPayload: "{\"brand\":\"visa\",\"expiryMonth\":\"03\",\"expiryYear\":\"22\",\"holderName\":\"Checkout Shopper PlaceHolder\",\"id\":\"BHKI4N2GMWVU2S56\",\"lastFour\":\"1234\",\"name\":\"VISA / Carte Bancaire\",\"networkTxReference\":\"123456789\",\"supportedShopperInteractions\":[\"Ecommerce\",\"ContAuth\"],\"type\":\"scheme\"}",
		DisplayValue:    "•••• 1234",
		PaymentProvider: Adyen,
		PaymentType:     CreditCard,
		IsPreferred:     true,
	}

	if expectedPaymentMethod1 != paymentMethods[0] {
		t.Fatalf("expected %v == %v\n", expectedPaymentMethod1, paymentMethods[0])
	}

}
