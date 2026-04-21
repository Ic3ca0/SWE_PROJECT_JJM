package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// aiRequest represents the frontend payload.
type aiRequest struct {
	Message string `json:"message"`
}

// aiResponse is what we return to the frontend.
type aiResponse struct {
	Reply string                 `json:"reply"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// handleAIChat reads the local food log (../data/food_log.json), composes
// a prompt, and forwards to configured Gemini endpoint. If GEMINI env vars
// are not set or the upstream call fails, a simple local analysis is returned.
func handleAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// read request body
	var req aiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// read data file
	dataPath := "../data/food_log.json"
	b, err := os.ReadFile(dataPath)
	if err != nil {
		http.Error(w, "Failed to read data file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// attempt to call Gemini if configured
	geminiKey := os.Getenv("GEMINI_API_KEY")
	geminiEndpoint := os.Getenv("GEMINI_ENDPOINT")

	// compose payload for upstream
	prompt := fmt.Sprintf("User message:\n%s\n\nFood log JSON:\n%s", req.Message, string(b))

	if geminiKey != "" && geminiEndpoint != "" {
		upstream := map[string]interface{}{
			"input":        prompt,
			"instructions": "Analyze the provided food log JSON and the user's message. Return a concise JSON object with keys: summary, recommendations.",
		}
		body, _ := json.Marshal(upstream)

		httpReq, err := http.NewRequest(http.MethodPost, geminiEndpoint, bytes.NewReader(body))
		if err == nil {
			httpReq.Header.Set("Authorization", "Bearer "+geminiKey)
			httpReq.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(httpReq)
			if err == nil {
				defer resp.Body.Close()
				respBytes, err := io.ReadAll(resp.Body)
				if err == nil {
					// best-effort: try to forward JSON response; otherwise wrap as text
					var parsed map[string]interface{}
					if json.Unmarshal(respBytes, &parsed) == nil {
						writeJSONResponse(w, parsed)
						return
					}
					// not JSON, return as text reply
					writeJSONResponse(w, aiResponse{Reply: string(respBytes)})
					return
				}
			}
			// fall through to local analysis on any error
		}
	}

	// local fallback analysis
	var entries []map[string]interface{}
	if err := json.Unmarshal(b, &entries); err != nil {
		// if parsing fails, return raw data message
		writeJSONResponse(w, aiResponse{Reply: "Could not parse log data."})
		return
	}

	// build summary: total calories per date and top foods by calories
	totals := map[string]float64{}
	foodCalories := map[string]float64{}
	foodCounts := map[string]int{}

	for _, e := range entries {
		date, _ := e["date"].(string)
		cal := 0.0
		switch v := e["calories"].(type) {
		case float64:
			cal = v
		case float32:
			cal = float64(v)
		case int:
			cal = float64(v)
		case int64:
			cal = float64(v)
		case string:
			// attempt parse
		}
		totals[date] += cal

		name := "unknown"
		if n, ok := e["name"].(string); ok {
			name = n
		}
		foodCalories[name] += cal
		foodCounts[name]++
	}

	// find top 3 foods by calories
	type kv struct {
		k string
		v float64
	}
	list := make([]kv, 0, len(foodCalories))
	for k, v := range foodCalories {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v > list[j].v })
	topN := 3
	top := []string{}
	for i := 0; i < len(list) && i < topN; i++ {
		top = append(top, fmt.Sprintf("%s (%.2f cal)", list[i].k, list[i].v))
	}

	// craft recommendations (simple heuristics)
	totalAll := 0.0
	for _, v := range totals {
		totalAll += v
	}

	recs := []string{}
	if totalAll > 2500 {
		recs = append(recs, "Your logged intake across dates is high — consider reducing portion sizes or swapping high-calorie items.")
	} else {
		recs = append(recs, "Overall intake appears moderate.")
	}
	if len(top) > 0 {
		recs = append(recs, "Top contributors: "+strings.Join(top, ", "))
	}
	recs = append(recs, "Aim to add more vegetables and whole grains; balance fats and proteins.")

	// include user message (if any) in reply
	reply := "Summary: \n"
	for d, v := range totals {
		reply += fmt.Sprintf("%s: %.2f cal\n", d, v)
	}
	reply += "\nRecommendations:\n- " + strings.Join(recs, "\n- ")
	if req.Message != "" {
		reply = "User question: " + req.Message + "\n\n" + reply
	}

	out := aiResponse{Reply: reply, Data: map[string]interface{}{"totals": totals, "top_foods": top}}
	writeJSONResponse(w, out)
}
