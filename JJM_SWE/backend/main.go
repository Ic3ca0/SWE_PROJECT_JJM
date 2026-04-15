package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Food struct {
	FoodID   string  `json:"food_id"`
	UserID   int     `json:"-"`
	Name     string  `json:"name"`
	KcalPerG float64 `json:"kcal_per_g"`
	FoodType string  `json:"food_type"`
}

type FoodEntry struct {
	EntryID  int     `json:"entry_id"`
	UserID   int     `json:"-"`
	Date     string  `json:"date"`
	FoodID   string  `json:"food_id"`
	Name     string  `json:"name"`
	Grams    float64 `json:"grams"`
	KcalPerG float64 `json:"kcal_per_g"`
	FoodType string  `json:"food_type"`
	Calories float64 `json:"calories"`
}

type CreateFoodRequest struct {
	Name         string  `json:"name"`
	Calories     float64 `json:"calories"`
	ServingGrams float64 `json:"serving_grams"`
	FoodType     string  `json:"food_type"`
}

type AddEntryRequest struct {
	Date   string  `json:"date"`
	FoodID string  `json:"food_id"`
	Grams  float64 `json:"grams"`
}

type GraphNode struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	Type  string  `json:"type"`
	Value float64 `json:"value"`
	Group string  `json:"group"`
}

type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type GraphResponse struct {
	DailyTotalCalories float64     `json:"daily_total_calories"`
	Nodes              []GraphNode `json:"nodes"`
	Links              []GraphLink `json:"links"`
}

type EntriesResponse struct {
	Date               string      `json:"date"`
	DailyTotalCalories float64     `json:"daily_total_calories"`
	Entries            []FoodEntry `json:"entries"`
}

// initialize server and routes
func main() {
	if err := initDB(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/api/register", withCORS(handleRegister))
	http.HandleFunc("/api/login", withCORS(handleLogin))
	http.HandleFunc("/api/logout", withCORS(handleLogout))
	http.HandleFunc("/api/me", withCORS(handleMe))

	http.HandleFunc("/api/foods", withCORS(withAuth(handleFoods)))
	http.HandleFunc("/api/foods/", withCORS(withAuth(handleDeleteFood)))
	http.HandleFunc("/api/entry", withCORS(withAuth(handleAddEntry)))
	http.HandleFunc("/api/entry/", withCORS(withAuth(handleDeleteEntry)))
	http.HandleFunc("/api/entries", withCORS(withAuth(handleGetEntries)))
	http.HandleFunc("/api/graph", withCORS(withAuth(handleGraph)))

	fs := http.FileServer(http.Dir("../frontend"))
	http.Handle("/", fs)

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func round2(n float64) float64 {
	return math.Round(n*100) / 100
}

// food crud handlers
func handleFoods(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)

	switch r.Method {
	case http.MethodGet:
		query := normalize(r.URL.Query().Get("query"))

		var foods []Food
		var err error
		if query != "" {
			foods, err = dbSearchFoods(userID, query)
		} else {
			foods, err = dbGetFoodsByUser(userID)
		}

		if err != nil {
			http.Error(w, "Failed to read foods", http.StatusInternalServerError)
			return
		}

		if foods == nil {
			foods = []Food{}
		}

		writeJSONResponse(w, foods)

	case http.MethodPost:
		var req CreateFoodRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.FoodType = normalize(req.FoodType)

		if req.Name == "" || req.Calories <= 0 || req.ServingGrams <= 0 || req.FoodType == "" {
			http.Error(w, "All fields are required and must be positive", http.StatusBadRequest)
			return
		}

		exists, err := dbFoodExistsByName(userID, req.Name)
		if err != nil {
			http.Error(w, "Failed to check food", http.StatusInternalServerError)
			return
		}
		if exists {
			http.Error(w, "Food already exists", http.StatusBadRequest)
			return
		}

		foodID, err := dbNextFoodID()
		if err != nil {
			http.Error(w, "Failed to generate food ID", http.StatusInternalServerError)
			return
		}

		kcalPerG := round2(req.Calories / req.ServingGrams)
		newFood := Food{
			FoodID:   foodID,
			UserID:   userID,
			Name:     req.Name,
			KcalPerG: kcalPerG,
			FoodType: req.FoodType,
		}

		if err := dbInsertFood(newFood); err != nil {
			http.Error(w, "Failed to save food", http.StatusInternalServerError)
			return
		}

		writeJSONResponse(w, newFood)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleDeleteFood(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := getUserIDFromRequest(r)
	foodID := strings.TrimPrefix(r.URL.Path, "/api/foods/")
	if foodID == "" {
		http.Error(w, "Food ID required", http.StatusBadRequest)
		return
	}

	found, err := dbDeleteFood(foodID, userID)
	if err != nil {
		http.Error(w, "Failed to delete food", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Food not found", http.StatusNotFound)
		return
	}

	writeJSONResponse(w, map[string]bool{"success": true})
}

// entry logging handlers
func handleAddEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := getUserIDFromRequest(r)

	var req AddEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Date) == "" {
		req.Date = time.Now().Format("2006-01-02")
	}

	if strings.TrimSpace(req.FoodID) == "" || req.Grams <= 0 {
		http.Error(w, "food_id and positive grams are required", http.StatusBadRequest)
		return
	}

	food, err := dbGetFoodByID(req.FoodID, userID)
	if err != nil {
		http.Error(w, "Food not found", http.StatusNotFound)
		return
	}

	newEntry := FoodEntry{
		UserID:   userID,
		Date:     req.Date,
		FoodID:   food.FoodID,
		Name:     food.Name,
		Grams:    req.Grams,
		KcalPerG: food.KcalPerG,
		FoodType: food.FoodType,
		Calories: round2(req.Grams * food.KcalPerG),
	}

	entryID, err := dbInsertEntry(newEntry)
	if err != nil {
		http.Error(w, "Failed to save entry", http.StatusInternalServerError)
		return
	}

	newEntry.EntryID = entryID
	writeJSONResponse(w, newEntry)
}

func handleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := getUserIDFromRequest(r)
	entryIDStr := strings.TrimPrefix(r.URL.Path, "/api/entry/")
	entryID, err := strconv.Atoi(entryIDStr)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	found, err := dbDeleteEntry(entryID, userID)
	if err != nil {
		http.Error(w, "Failed to delete entry", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	writeJSONResponse(w, map[string]bool{"success": true})
}

func handleGetEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := getUserIDFromRequest(r)
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	entries, err := dbGetEntriesByDate(userID, date)
	if err != nil {
		http.Error(w, "Failed to read entries", http.StatusInternalServerError)
		return
	}

	if entries == nil {
		entries = []FoodEntry{}
	}

	total := 0.0
	for _, e := range entries {
		total += e.Calories
	}

	resp := EntriesResponse{
		Date:               date,
		DailyTotalCalories: round2(total),
		Entries:            entries,
	}

	writeJSONResponse(w, resp)
}

// graph data builder
func handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := getUserIDFromRequest(r)
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	entries, err := dbGetEntriesByDate(userID, date)
	if err != nil {
		http.Error(w, "Failed to read entries", http.StatusInternalServerError)
		return
	}

	if entries == nil {
		entries = []FoodEntry{}
	}

	graph := buildGraph(entries)
	writeJSONResponse(w, graph)
}

func buildGraph(entries []FoodEntry) GraphResponse {
	if len(entries) == 0 {
		return GraphResponse{
			DailyTotalCalories: 0,
			Nodes:              []GraphNode{},
			Links:              []GraphLink{},
		}
	}

	groupTotals := map[string]float64{}
	for _, e := range entries {
		groupTotals[e.FoodType] += e.Calories
	}

	total := 0.0
	for _, v := range groupTotals {
		total += v
	}
	total = round2(total)

	nodes := []GraphNode{}
	links := []GraphLink{}

	for group, calories := range groupTotals {
		percent := 0.0
		if total > 0 {
			percent = (calories / total) * 100
		}

		nodes = append(nodes, GraphNode{
			ID:    group,
			Label: fmt.Sprintf("%s\n%.2f cal\n%.1f%%", titleCase(group), round2(calories), percent),
			Type:  "group",
			Value: round2(calories),
			Group: group,
		})
	}

	for _, e := range entries {
		nodeID := fmt.Sprintf("entry-%d", e.EntryID)
		nodes = append(nodes, GraphNode{
			ID:    nodeID,
			Label: fmt.Sprintf("%s\n%.0fg\n%.2f cal", titleCase(e.Name), e.Grams, e.Calories),
			Type:  "food",
			Value: round2(e.Calories),
			Group: e.FoodType,
		})
		links = append(links, GraphLink{
			Source: e.FoodType,
			Target: nodeID,
		})
	}

	return GraphResponse{
		DailyTotalCalories: total,
		Nodes:              nodes,
		Links:              links,
	}
}

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
