package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// setup in memory test database
func setupTestDB(t *testing.T) {
	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.Exec("PRAGMA foreign_keys = ON")
	if err := createTables(); err != nil {
		t.Fatal(err)
	}
}

func registerTestUser(t *testing.T) {
	body := strings.NewReader(`{"username":"testuser","password":"testpass"}`)
	req := httptest.NewRequest("POST", "/api/register", body)
	rec := httptest.NewRecorder()
	handleRegister(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register failed: %d %s", rec.Code, rec.Body.String())
	}
}

func loginTestUser(t *testing.T) *http.Cookie {
	body := strings.NewReader(`{"username":"testuser","password":"testpass"}`)
	req := httptest.NewRequest("POST", "/api/login", body)
	rec := httptest.NewRecorder()
	handleLogin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatal("no session cookie returned")
	return nil
}

func TestRegisterAndLogin(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	registerTestUser(t)

	body := strings.NewReader(`{"username":"testuser","password":"testpass"}`)
	req := httptest.NewRequest("POST", "/api/login", body)
	rec := httptest.NewRecorder()
	handleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rec.Code)
	}

	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected session cookie after login")
	}
}

func TestBadLogin(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	registerTestUser(t)

	body := strings.NewReader(`{"username":"testuser","password":"wrongpass"}`)
	req := httptest.NewRequest("POST", "/api/login", body)
	rec := httptest.NewRecorder()
	handleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d", rec.Code)
	}
}

// api endpoint tests
func TestFoodCRUD(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	registerTestUser(t)
	session := loginTestUser(t)

	body := strings.NewReader(`{"name":"Chicken","calories":165,"serving_grams":100,"food_type":"protein"}`)
	req := httptest.NewRequest("POST", "/api/foods", body)
	req.AddCookie(session)
	rec := httptest.NewRecorder()
	handleFoods(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("create food failed: %d %s", rec.Code, rec.Body.String())
	}

	var food Food
	json.NewDecoder(rec.Body).Decode(&food)
	if food.Name != "Chicken" {
		t.Errorf("expected Chicken got %s", food.Name)
	}
	if food.FoodID == "" {
		t.Error("expected a food_id")
	}

	req = httptest.NewRequest("GET", "/api/foods", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	handleFoods(rec, req)

	var foods []Food
	json.NewDecoder(rec.Body).Decode(&foods)
	if len(foods) != 35 {
		t.Errorf("expected 35 foods (34 seeded + 1 new) got %d", len(foods))
	}
}

func TestEntryLogging(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	registerTestUser(t)
	session := loginTestUser(t)

	body := strings.NewReader(`{"name":"Rice","calories":130,"serving_grams":100,"food_type":"grains"}`)
	req := httptest.NewRequest("POST", "/api/foods", body)
	req.AddCookie(session)
	rec := httptest.NewRecorder()
	handleFoods(rec, req)

	var food Food
	json.NewDecoder(rec.Body).Decode(&food)

	entryBody := strings.NewReader(`{"food_id":"` + food.FoodID + `","grams":200,"date":"2026-04-15"}`)
	req = httptest.NewRequest("POST", "/api/entry", entryBody)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	handleAddEntry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("add entry failed: %d %s", rec.Code, rec.Body.String())
	}

	var entry FoodEntry
	json.NewDecoder(rec.Body).Decode(&entry)
	if entry.Calories != 260.0 {
		t.Errorf("expected 260 calories got %.2f", entry.Calories)
	}

	req = httptest.NewRequest("GET", "/api/entries?date=2026-04-15", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	handleGetEntries(rec, req)

	var resp EntriesResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Entries) != 1 {
		t.Errorf("expected 1 entry got %d", len(resp.Entries))
	}
	if resp.DailyTotalCalories != 260.0 {
		t.Errorf("expected 260 total got %.2f", resp.DailyTotalCalories)
	}
}

func TestGraphData(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	registerTestUser(t)
	session := loginTestUser(t)

	body := strings.NewReader(`{"name":"Dragon Fruit","calories":52,"serving_grams":100,"food_type":"fruit"}`)
	req := httptest.NewRequest("POST", "/api/foods", body)
	req.AddCookie(session)
	rec := httptest.NewRecorder()
	handleFoods(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("create food failed: %d %s", rec.Code, rec.Body.String())
	}

	var food Food
	json.NewDecoder(rec.Body).Decode(&food)

	entryBody := strings.NewReader(`{"food_id":"` + food.FoodID + `","grams":150,"date":"2026-04-15"}`)
	req = httptest.NewRequest("POST", "/api/entry", entryBody)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	handleAddEntry(rec, req)

	req = httptest.NewRequest("GET", "/api/graph?date=2026-04-15", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	handleGraph(rec, req)

	var graph GraphResponse
	json.NewDecoder(rec.Body).Decode(&graph)
	if len(graph.Nodes) != 2 {
		t.Errorf("expected 2 nodes (1 group + 1 food) got %d", len(graph.Nodes))
	}
	if len(graph.Links) != 1 {
		t.Errorf("expected 1 link got %d", len(graph.Links))
	}
}

func TestUnauthorizedAccess(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/me", nil)
	rec := httptest.NewRecorder()
	handleMe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d", rec.Code)
	}
}

func TestDuplicateFood(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	registerTestUser(t)
	session := loginTestUser(t)

	body := strings.NewReader(`{"name":"Lamb Chops","calories":271,"serving_grams":100,"food_type":"protein"}`)
	req := httptest.NewRequest("POST", "/api/foods", body)
	req.AddCookie(session)
	rec := httptest.NewRecorder()
	handleFoods(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("first create failed: %d", rec.Code)
	}

	body = strings.NewReader(`{"name":"Lamb Chops","calories":271,"serving_grams":100,"food_type":"protein"}`)
	req = httptest.NewRequest("POST", "/api/foods", body)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	handleFoods(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate got %d", rec.Code)
	}
}