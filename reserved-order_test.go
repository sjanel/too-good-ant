package main

import (
	"os"
	"testing"
)

const (
	kExampleReservedOrder = "data/test/example_reserved_order.json"
)

func TestReservedOrderStandardResponse(t *testing.T) {
	responseBody, err := os.ReadFile(kExampleReservedOrder)
	if err != nil {
		t.Fatalf("error reading file %v", kExampleReservedOrder)
	}
	reservedOrder, err := NewReservedOrderFromCreateOrder(string(responseBody))
	if err != nil {
		t.Fatalf("error in NewReservedOrderFromCreateOrder")
	}

	expectedReservedOrder := ReservedOrder{
		Id:       "order_id12345",
		StoreId:  "item_id12345",
		Quantity: 1,
	}

	if reservedOrder != expectedReservedOrder {
		t.Fatalf("expected reserved order %v, got %v", expectedReservedOrder, reservedOrder)
	}
}