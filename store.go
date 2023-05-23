package main

import (
	"encoding/json"
	"fmt"
	"math"
)

type Price struct {
	Amount       int
	NbDecimals   int
	CurrencyCode string
}

func (p Price) String() string {
	return fmt.Sprintf("%v %v", float64(p.Amount)/math.Pow(10, float64(p.NbDecimals)), p.CurrencyCode)
}

type Store struct {
	Name   string
	Id     string
	Rating float64
	Price  Price
}

func (s *Store) String() string {
	return fmt.Sprintf("%v, rated %v, price %v", s.Name, s.Rating, s.Price)
}

func CreateStoresFromListStoresResponse(responseBody string) ([]Store, error) {
	if len(responseBody) == 0 {
		return []Store{}, nil
	}

	var parsedItems map[string][]map[string]interface{}
	err := json.Unmarshal([]byte(responseBody), &parsedItems)
	if err != nil {
		glog.Printf("full response: %v\n", responseBody)
		return []Store{}, fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	items := parsedItems["items"]

	stores := make([]Store, len(items))

	for itemPos, item := range items {
		itemParsed := item["item"].(map[string]interface{})

		priceIncludingTaxes := itemParsed["price_including_taxes"].(map[string]interface{})

		stores[itemPos].Id = itemParsed["item_id"].(string)

		stores[itemPos].Price.Amount = int(priceIncludingTaxes["minor_units"].(float64))
		stores[itemPos].Price.CurrencyCode = priceIncludingTaxes["code"].(string)
		stores[itemPos].Price.NbDecimals = int(priceIncludingTaxes["decimals"].(float64))

		rating, hasRating := itemParsed["average_overall_rating"]
		if hasRating {
			stores[itemPos].Rating = rating.(map[string]interface{})["average_overall_rating"].(float64)
		}

		storeParsed := item["store"].(map[string]interface{})

		stores[itemPos].Name = storeParsed["store_name"].(string)
	}

	return stores, nil
}
