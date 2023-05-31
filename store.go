package main

import (
	"encoding/json"
	"fmt"
)

type Store struct {
	Name          string
	Id            string
	Rating        float64
	Price         Price
	AvailableBags int
}

func (s *Store) String() string {
	return fmt.Sprintf("%v, rated %v, price %v, %v available bags", s.Name, s.Rating, s.Price, s.AvailableBags)
}

func NewStoresFromListStoresResponse(responseBody []byte) ([]Store, error) {
	if len(responseBody) == 0 {
		return []Store{}, nil
	}

	var parsedItems map[string][]map[string]interface{}
	err := json.Unmarshal(responseBody, &parsedItems)
	if err != nil {
		glog.Printf("full response: %v\n", string(responseBody))
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
		stores[itemPos].AvailableBags = int(item["items_available"].(float64))
	}

	return stores, nil
}
