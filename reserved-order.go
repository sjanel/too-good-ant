package main

import (
	"encoding/json"
	"fmt"
)

type ReservedOrder struct {
	Id       string
	StoreId  string
	Quantity int
}

func (o *ReservedOrder) String() string {
	return fmt.Sprintf("Order # %v in store %v with %v bags", o.Id, o.StoreId, o.Quantity)
}

func NewReservedOrderFromCreateOrder(responseBody string) (ReservedOrder, error) {
	var reservedOrder ReservedOrder
	if len(responseBody) == 0 {
		return reservedOrder, nil
	}

	var parsedReservedOrder map[string]interface{}
	err := json.Unmarshal([]byte(responseBody), &parsedReservedOrder)
	if err != nil {
		glog.Printf("full response: %v\n", responseBody)
		return reservedOrder, fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	state := parsedReservedOrder["state"].(string)
	if state != "SUCCESS" {
		return reservedOrder, fmt.Errorf("reserved order state %v is not OK", state)
	}

	reservedOrderData := parsedReservedOrder["order"].(map[string]interface{})

	reservedOrder.Id = reservedOrderData["id"].(string)
	reservedOrder.StoreId = reservedOrderData["item_id"].(string)
	reservedOrder.Quantity = int(reservedOrderData["order_line"].(map[string]interface{})["quantity"].(float64))

	return reservedOrder, nil
}
