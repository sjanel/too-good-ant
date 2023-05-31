package tga

import (
	"encoding/json"
	"fmt"
	"time"
)

type PickupDetails struct {
	Address string
	FromGMT time.Time
	ToGMT   time.Time
}

func (p *PickupDetails) String() string {
	return fmt.Sprintf("%v between [%v, %v]", p.Address, p.FromGMT, p.ToGMT)
}

type Order struct {
	StoreName     string
	StoreId       string
	State         string
	Id            string
	PickupDetails PickupDetails
	Rating        float64
	Price         Price
	Quantity      int
}

func (o *Order) String() string {
	return fmt.Sprintf("Order # %v, Store # %v, with %v bags to pick at %v", o.Id, o.StoreId, o.Quantity, o.PickupDetails)
}

func NewOrdersFromListOrdersResponse(responseBody []byte) ([]Order, error) {
	if len(responseBody) == 0 {
		return []Order{}, nil
	}

	var parsedOrders map[string]interface{}
	err := json.Unmarshal(responseBody, &parsedOrders)
	if err != nil {
		glog.Printf("full response: %v\n", string(responseBody))
		return []Order{}, fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	items := parsedOrders["orders"].([]interface{})

	orders := make([]Order, len(items))

	for itemPos, item := range items {
		parsedItem := item.(map[string]interface{})
		orders[itemPos].StoreName = parsedItem["store_name"].(string)
		orders[itemPos].StoreId = parsedItem["store_id"].(string)
		orders[itemPos].State = parsedItem["state"].(string)
		orders[itemPos].Id = parsedItem["order_id"].(string)
		orders[itemPos].Price = NewPrice(parsedItem["price_including_taxes"].(map[string]interface{}))
		orders[itemPos].Quantity = int(parsedItem["quantity"].(float64))

		orders[itemPos].PickupDetails.Address = parsedItem["pickup_location"].(map[string]interface{})["address"].(map[string]interface{})["address_line"].(string)

		pickupInterval := parsedItem["pickup_interval"].(map[string]interface{})

		orders[itemPos].PickupDetails.FromGMT, err = time.Parse(time.RFC3339, pickupInterval["start"].(string))
		if err != nil {
			return []Order{}, fmt.Errorf("error in time.Parse: %w", err)
		}
		orders[itemPos].PickupDetails.ToGMT, err = time.Parse(time.RFC3339, pickupInterval["end"].(string))
		if err != nil {
			return []Order{}, fmt.Errorf("error in time.Parse: %w", err)
		}
	}

	return orders, nil
}
