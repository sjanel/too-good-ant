package tga

import (
	"encoding/json"
	"fmt"
)

type PaymentProvider int

const (
	Cash PaymentProvider = iota
	Braintree
	Adyen
	Satispay
)

func (p PaymentProvider) String() string {
	switch p {
	case Cash:
		return "CASH"
	case Braintree:
		return "BRAINTREE"
	case Adyen:
		return "ADYEN"
	case Satispay:
		return "SATISPAY"
	}
	return "unknown"
}

func NewPaymentProvider(str string) (PaymentProvider, error) {
	if str == "CASH" {
		return Cash, nil
	}
	if str == "BRAINTREE" {
		return Braintree, nil
	}
	if str == "ADYEN" {
		return Adyen, nil
	}
	if str == "SATISPAY" {
		return Satispay, nil
	}
	return -1, fmt.Errorf("unknown payment provider %v", str)
}

func (p PaymentProvider) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *PaymentProvider) UnmarshalJSON(b []byte) error {
	paymentProvider, err := NewPaymentProvider(string(b[1 : len(b)-1]))
	if err != nil {
		return fmt.Errorf("error from NewPaymentProvider: %w", err)
	}
	*p = paymentProvider
	return nil
}

func (p PaymentProvider) GetAuthorizationPayloadType() string {
	// [adyenAuthorizationPayload, braintreeAuthorizationPayload, cashAuthorizationPayload, charityAuthorizationPayload, fakeDoorAuthorizationPayload, satispayAuthorizationPayload, voucherAuthorizationPayload]
	switch p {
	case Cash:
		return "cashAuthorizationPayload"
	case Braintree:
		return "braintreeAuthorizationPayload"
	case Adyen:
		return "adyenAuthorizationPayload"
	case Satispay:
		return "satispayAuthorizationPayload"
	}
	return "unknown"
}

type PaymentType int

const (
	CreditCard PaymentType = iota
	GooglePay
	BcMcMobile
	BcMcCard
	Vipps
	Twint
	MbWay
	Swish
	Blik
	Venmo
	FakeDoor
	PayPal
	SoFort
)

func (p PaymentType) String() string {
	switch p {
	case CreditCard:
		return "CREDITCARD"
	case GooglePay:
		return "GOOGLEPAY"
	case BcMcMobile:
		return "BCMCMOBILE"
	case BcMcCard:
		return "BCMCCARD"
	case Vipps:
		return "VIPPS"
	case Twint:
		return "TWINT"
	case MbWay:
		return "MBWAY"
	case Swish:
		return "SWISH"
	case Blik:
		return "BLIK"
	case Venmo:
		return "VENMO"
	case FakeDoor:
		return "FAKE_DOOR"
	case PayPal:
		return "PAYPAL"
	case SoFort:
		return "SOFORT"
	}
	return "unknown"
}

func NewPaymentType(str string) (PaymentType, error) {
	if str == "CREDITCARD" {
		return CreditCard, nil
	}
	if str == "GOOGLEPAY" {
		return GooglePay, nil
	}
	if str == "BCMCMOBILE" {
		return BcMcMobile, nil
	}
	if str == "BCMCCARD" {
		return BcMcCard, nil
	}
	if str == "VIPPS" {
		return Vipps, nil
	}
	if str == "TWINT" {
		return Twint, nil
	}
	if str == "MBWAY" {
		return MbWay, nil
	}
	if str == "SWISH" {
		return Swish, nil
	}
	if str == "BLIK" {
		return Blik, nil
	}
	if str == "VENMO" {
		return Venmo, nil
	}
	if str == "FAKE_DOOR" {
		return FakeDoor, nil
	}
	if str == "PAYPAL" {
		return PayPal, nil
	}
	if str == "SOFORT" {
		return SoFort, nil
	}
	return -1, fmt.Errorf("unknown payment type %v", str)
}

func (p PaymentType) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *PaymentType) UnmarshalJSON(b []byte) error {
	paymentType, err := NewPaymentType(string(b[1 : len(b)-1]))
	if err != nil {
		return fmt.Errorf("error from NewPaymentType: %w", err)
	}
	*p = paymentType
	return nil
}
