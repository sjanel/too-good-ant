package main

import (
	"encoding/json"
	"fmt"
)

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

		stores[itemPos].Id = itemParsed["item_id"].(string)

		stores[itemPos].Price = NewPrice(itemParsed["price_including_taxes"].(map[string]interface{}))

		rating, hasRating := itemParsed["average_overall_rating"]
		if hasRating {
			stores[itemPos].Rating = rating.(map[string]interface{})["average_overall_rating"].(float64)
		}

		storeParsed := item["store"].(map[string]interface{})

		stores[itemPos].Name = storeParsed["store_name"].(string)
	}

	return stores, nil
}
