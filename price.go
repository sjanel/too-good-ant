package main

import (
	"fmt"
	"math"
)

type Price struct {
	Amount       int
	NbDecimals   int
	CurrencyCode string
}

func (p Price) FloatAmount() float64 {
	return float64(p.Amount) / math.Pow(10, float64(p.NbDecimals))
}

func (p Price) String() string {
	return fmt.Sprintf("%v %v", p.FloatAmount(), p.CurrencyCode)
}

func NewPrice(priceIncludingTaxes map[string]interface{}) Price {
	return Price{
		Amount:       int(priceIncludingTaxes["minor_units"].(float64)),
		CurrencyCode: priceIncludingTaxes["code"].(string),
		NbDecimals:   int(priceIncludingTaxes["decimals"].(float64)),
	}
}
