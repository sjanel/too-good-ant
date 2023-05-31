package tga

import (
	"os"
	"reflect"
	"testing"
	"time"
)

const (
	kExampleOrderPath = "testdata/example_order.json"
)

func TestOrderEmptyResponse(t *testing.T) {
	orders, err := NewOrdersFromListOrdersResponse([]byte{})
	if len(orders) > 0 || err != nil {
		t.Fatalf("expected empty list of orders for empty response")
	}
}

func TestOrderStandardResponse(t *testing.T) {
	responseBody, err := os.ReadFile(kExampleOrderPath)
	if err != nil {
		t.Fatalf("error reading file %v", kExampleOrderPath)
	}
	orders, err := NewOrdersFromListOrdersResponse(responseBody)
	if err != nil {
		t.Fatalf("error in NewOrdersFromListOrdersResponse: %v", err)
	}

	nbExpectedOrders := 1

	if len(orders) != nbExpectedOrders {
		t.Fatalf("expected %v orders, got %v", nbExpectedOrders, len(orders))
	}

	order1 := orders[0]

	expectedFromGMT, err := time.Parse(time.RFC3339, "2023-05-23T13:00:00Z")
	if err != nil {
		t.Fatalf("error in time.Parse: %v", err)
	}

	expectedToGMT, err := time.Parse(time.RFC3339, "2023-05-23T15:00:00Z")
	if err != nil {
		t.Fatalf("error in time.Parse: %v", err)
	}

	expectedOrder1 := Order{
		StoreName: "My Store name",
		StoreId:   "32949",
		State:     "ACTIVE",
		Id:        "gkmfvwixjf0",
		PickupDetails: PickupDetails{
			Address: "Piazza del Colosseo, 1, 00184 Roma RM, Italia",
			FromGMT: expectedFromGMT,
			ToGMT:   expectedToGMT,
		},
		Rating: 0,
		Price: Price{
			Amount:       399,
			NbDecimals:   2,
			CurrencyCode: "EUR",
		},
		Quantity: 1,
	}

	if order1.StoreName != expectedOrder1.StoreName {
		t.Fatalf("expected store name %v, got %v", expectedOrder1.StoreName, order1.StoreName)
	}
	if order1.StoreId != expectedOrder1.StoreId {
		t.Fatalf("expected store id %v, got %v", expectedOrder1.StoreId, order1.StoreId)
	}
	if order1.State != expectedOrder1.State {
		t.Fatalf("expected state %v, got %v", expectedOrder1.State, order1.State)
	}
	if order1.Id != expectedOrder1.Id {
		t.Fatalf("expected id %v, got %v", expectedOrder1.Id, order1.Id)
	}
	if order1.PickupDetails != expectedOrder1.PickupDetails {
		t.Fatalf("expected pickup details %v, got %v", expectedOrder1.PickupDetails, order1.PickupDetails)
	}
	if order1.Rating != expectedOrder1.Rating {
		t.Fatalf("expected rating %v, got %v", expectedOrder1.Rating, order1.Rating)
	}
	if order1.Price != expectedOrder1.Price {
		t.Fatalf("expected price %v, got %v", expectedOrder1.Price, order1.Price)
	}
	if order1.Quantity != expectedOrder1.Quantity {
		t.Fatalf("expected quantity %v, got %v", expectedOrder1.Quantity, order1.Quantity)
	}
}

func TestEqual(t *testing.T) {
	price := Price{
		Amount:       399,
		NbDecimals:   2,
		CurrencyCode: "EUR",
	}
	store1 := Store{
		Name:   "store 1",
		Id:     "23",
		Rating: 3.5,
		Price:  price,
	}
	store2 := Store{
		Name:   "store 1",
		Id:     "23",
		Rating: 3.5,
		Price:  price,
	}
	store3 := Store{
		Name:   "store 1",
		Id:     "24",
		Rating: 3.5,
		Price:  price,
	}

	if store1 != store2 {
		t.Fatalf("stores should be compared by id (expected %v == %v)\n", store1, store2)
	}
	if store1 == store3 {
		t.Fatalf("stores should be compared by id (expected %v != %v)\n", store1, store3)
	}

	stores1 := []Store{store1, store2}
	stores2 := []Store{store2, store1}
	stores3 := []Store{store2, store3}

	if !reflect.DeepEqual(stores1, stores2) {
		t.Fatalf("stores should be compared by id (expected %v == %v)\n", stores1, stores2)
	}
	if reflect.DeepEqual(stores1, stores3) {
		t.Fatalf("stores should be compared by id (expected %v != %v)\n", stores1, stores3)
	}
}
