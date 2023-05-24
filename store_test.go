package main

import (
	"os"
	"reflect"
	"testing"
)

const (
	kExampleStorePath = "data/example_list.json"
)

func TestStoreEmptyResponse(t *testing.T) {
	stores, err := CreateStoresFromListStoresResponse("")
	if len(stores) > 0 || err != nil {
		t.Fatalf("expected empty list of stores for empty response")
	}
}

func TestStoreStandardResponse(t *testing.T) {
	responseBody, err := os.ReadFile(kExampleStorePath)
	if err != nil {
		t.Fatalf("error reading file %v", kExampleStorePath)
	}
	stores, err := CreateStoresFromListStoresResponse(string(responseBody))
	if err != nil {
		t.Fatalf("error in CreateStoresFromListStoresResponse")
	}
	if len(stores) == 0 {
		t.Fatalf("expected non empty list of stores for empty response")
	}

	nbExpectedStores := 11

	if len(stores) != nbExpectedStores {
		t.Fatalf("expected %v stores, got %v", nbExpectedStores, len(stores))
	}

	store1 := stores[0]
	expectedStore1 := Store{
		Name:   "Ennao",
		Id:     "523087",
		Rating: 0,
		Price: Price{
			Amount:       399,
			NbDecimals:   2,
			CurrencyCode: "EUR",
		},
	}

	if store1.Name != expectedStore1.Name {
		t.Fatalf("expected name %v, got %v", expectedStore1.Name, store1.Name)
	}
	if store1.Id != expectedStore1.Id {
		t.Fatalf("expected id %v, got %v", expectedStore1.Id, store1.Id)
	}
	if store1.Rating != expectedStore1.Rating {
		t.Fatalf("expected rating %v, got %v", expectedStore1.Rating, store1.Rating)
	}
	if store1.Price != expectedStore1.Price {
		t.Fatalf("expected price %v, got %v", expectedStore1.Price, store1.Price)
	}
}

func TestStoreEqual(t *testing.T) {
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
