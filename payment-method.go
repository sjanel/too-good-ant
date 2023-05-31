package main

import (
	"encoding/json"
	"fmt"
)

type PaymentMethod struct {
	Id                string
	InternalType      string
	AdyenApiPayload   string
	DisplayValue      string
	SavePaymentMethod string // TODO: check what it is
	PaymentProvider   PaymentProvider
	PaymentType       PaymentType
	IsPreferred       bool
}

func (p PaymentMethod) String() string {
	return fmt.Sprintf("%v%v, isPreferred=%v", p.PaymentType, p.DisplayValue, p.IsPreferred)
}

func NewPaymentMethodsFromPaymentMethodsResponse(responseBody []byte) ([]PaymentMethod, error) {
	var parsedPaymentMethods map[string]interface{}
	err := json.Unmarshal(responseBody, &parsedPaymentMethods)
	if err != nil {
		glog.Printf("full response: %v\n", string(responseBody))
		return []PaymentMethod{}, fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	items := parsedPaymentMethods["payment_methods"].([]interface{})

	paymentMethods := make([]PaymentMethod, len(items))

	for itemPos, item := range items {
		parsedPaymentMethod := item.(map[string]interface{})

		// Warning, it seems that most fields are optional

		id, hasId := parsedPaymentMethod["identifier"]
		if hasId {
			paymentMethods[itemPos].Id = id.(string)
		}

		internalType, hasInternalType := parsedPaymentMethod["type"]
		if hasInternalType {
			paymentMethods[itemPos].InternalType = internalType.(string)
		}

		aydenApiPayload, hasAydenApiPayload := parsedPaymentMethod["adyen_api_payload"]
		if hasAydenApiPayload {
			paymentMethods[itemPos].AdyenApiPayload = aydenApiPayload.(string)
		}

		displayValue, hasDisplayValue := parsedPaymentMethod["display_value"]
		if hasDisplayValue {
			paymentMethods[itemPos].DisplayValue = displayValue.(string)
		}

		savePaymentMethod, hasSavePaymentMethod := parsedPaymentMethod["save_payment_method"]
		if hasSavePaymentMethod {
			paymentMethods[itemPos].SavePaymentMethod = savePaymentMethod.(string)
		}

		paymentProvider, hasPaymentProvider := parsedPaymentMethod["payment_provider"]
		if hasPaymentProvider {
			paymentMethods[itemPos].PaymentProvider, err = NewPaymentProvider(paymentProvider.(string))
			if err != nil {
				return paymentMethods, fmt.Errorf("error from NewPaymentProvider: %w", err)
			}
		}

		paymentType, hasPaymentType := parsedPaymentMethod["payment_type"]
		if hasPaymentType {
			paymentMethods[itemPos].PaymentType, err = NewPaymentType(paymentType.(string))
			if err != nil {
				return paymentMethods, fmt.Errorf("error from NewPaymentType: %w", err)
			}
		}

		isPreferred, hasIsPreferred := parsedPaymentMethod["preferred"]
		if hasIsPreferred {
			paymentMethods[itemPos].IsPreferred = isPreferred.(bool)
		}
	}

	return paymentMethods, nil
}
