package main

import (
	"os"
	"reflect"
	"testing"
)

const (
	kExamplePath = "data/example_list.json"
)

func TestEmptyResponse(t *testing.T) {
	stores, err := CreateStoresFromListStoresResponse("")
	if len(stores) > 0 || err != nil {
		t.Fatalf("expected empty list of stores for empty response")
	}
}

func TestResponse(t *testing.T) {
	responseBody, err := os.ReadFile(kExamplePath)
	if err != nil {
		t.Fatalf("error reading file %v", kExamplePath)
	}
	stores, err := CreateStoresFromListStoresResponse(string(responseBody))
	if err != nil {
		t.Fatalf("error in CreateStoresFromListStoresResponse")
	}
	if len(stores) == 0 {
		t.Fatalf("expected non empty list of stores for empty response")
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
