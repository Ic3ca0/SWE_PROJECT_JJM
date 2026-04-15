package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB() error {
	dataDir := "../data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	var err error
	db, err = sql.Open("sqlite", "../data/foodgraph.db")
	if err != nil {
		return err
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	return createTables()
}

// setup all sqlite tables
func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS foods (
		food_id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		kcal_per_g REAL NOT NULL,
		food_type TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS food_log (
		entry_id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		food_id TEXT NOT NULL,
		name TEXT NOT NULL,
		grams REAL NOT NULL,
		kcal_per_g REAL NOT NULL,
		food_type TEXT NOT NULL,
		calories REAL NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	`
	_, err := db.Exec(schema)
	return err
}

func dbCreateUser(username, passwordHash string) error {
	_, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, passwordHash)
	return err
}

func dbGetUserByUsername(username string) (int, string, error) {
	var id int
	var hash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&id, &hash)
	return id, hash, err
}

func dbGetUsernameByID(userID int) (string, error) {
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	return username, err
}

func dbCreateSession(token string, userID int, expiresAt string) error {
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)", token, userID, expiresAt)
	return err
}

func dbGetSessionUserID(token string) (int, error) {
	var userID int
	err := db.QueryRow("SELECT user_id FROM sessions WHERE token = ? AND expires_at > datetime('now')", token).Scan(&userID)
	return userID, err
}

func dbDeleteSession(token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// food and entry database operations
func dbNextFoodID() (string, error) {
	var maxNum int
	err := db.QueryRow(`SELECT COALESCE(MAX(CAST(REPLACE(food_id, 'food-', '') AS INTEGER)), 0) FROM foods`).Scan(&maxNum)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("food-%d", maxNum+1), nil
}

func dbInsertFood(f Food) error {
	_, err := db.Exec(
		"INSERT INTO foods (food_id, user_id, name, kcal_per_g, food_type) VALUES (?, ?, ?, ?, ?)",
		f.FoodID, f.UserID, f.Name, f.KcalPerG, f.FoodType,
	)
	return err
}

func dbGetFoodsByUser(userID int) ([]Food, error) {
	rows, err := db.Query(
		"SELECT food_id, user_id, name, kcal_per_g, food_type FROM foods WHERE user_id = ?", userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foods []Food
	for rows.Next() {
		var f Food
		if err := rows.Scan(&f.FoodID, &f.UserID, &f.Name, &f.KcalPerG, &f.FoodType); err != nil {
			return nil, err
		}
		foods = append(foods, f)
	}
	return foods, nil
}

func dbSearchFoods(userID int, query string) ([]Food, error) {
	q := "%" + strings.ToLower(query) + "%"
	rows, err := db.Query(
		"SELECT food_id, user_id, name, kcal_per_g, food_type FROM foods WHERE user_id = ? AND (LOWER(name) LIKE ? OR LOWER(food_type) LIKE ?)",
		userID, q, q,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foods []Food
	for rows.Next() {
		var f Food
		if err := rows.Scan(&f.FoodID, &f.UserID, &f.Name, &f.KcalPerG, &f.FoodType); err != nil {
			return nil, err
		}
		foods = append(foods, f)
	}
	return foods, nil
}

func dbGetFoodByID(foodID string, userID int) (*Food, error) {
	var f Food
	err := db.QueryRow(
		"SELECT food_id, user_id, name, kcal_per_g, food_type FROM foods WHERE food_id = ? AND user_id = ?",
		foodID, userID,
	).Scan(&f.FoodID, &f.UserID, &f.Name, &f.KcalPerG, &f.FoodType)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func dbFoodExistsByName(userID int, name string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM foods WHERE user_id = ? AND LOWER(name) = LOWER(?)",
		userID, strings.TrimSpace(name),
	).Scan(&count)
	return count > 0, err
}

func dbDeleteFood(foodID string, userID int) (bool, error) {
	result, err := db.Exec("DELETE FROM foods WHERE food_id = ? AND user_id = ?", foodID, userID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// entry log operations
func dbInsertEntry(e FoodEntry) (int, error) {
	result, err := db.Exec(
		`INSERT INTO food_log (user_id, date, food_id, name, grams, kcal_per_g, food_type, calories)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.UserID, e.Date, e.FoodID, e.Name, e.Grams, e.KcalPerG, e.FoodType, e.Calories,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int(id), err
}

func dbGetEntriesByDate(userID int, date string) ([]FoodEntry, error) {
	rows, err := db.Query(
		`SELECT entry_id, user_id, date, food_id, name, grams, kcal_per_g, food_type, calories
		FROM food_log WHERE user_id = ? AND date = ?`,
		userID, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []FoodEntry
	for rows.Next() {
		var e FoodEntry
		if err := rows.Scan(&e.EntryID, &e.UserID, &e.Date, &e.FoodID, &e.Name, &e.Grams, &e.KcalPerG, &e.FoodType, &e.Calories); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func dbDeleteEntry(entryID int, userID int) (bool, error) {
	result, err := db.Exec("DELETE FROM food_log WHERE entry_id = ? AND user_id = ?", entryID, userID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// seed common foods for new users
func dbSeedFoodsForUser(userID int) error {
	seeds := []struct {
		name     string
		kcal     float64
		foodType string
	}{
		{"Chicken Breast", 1.65, "protein"},
		{"Steak", 2.71, "protein"},
		{"Salmon", 2.08, "protein"},
		{"Eggs", 1.55, "protein"},
		{"Ground Beef", 2.54, "protein"},
		{"Turkey Breast", 1.35, "protein"},
		{"Shrimp", 0.99, "protein"},
		{"Tofu", 0.76, "protein"},
		{"Tuna", 1.32, "protein"},
		{"White Rice", 1.30, "grains"},
		{"Brown Rice", 1.12, "grains"},
		{"Pasta", 1.31, "grains"},
		{"Bread", 2.65, "grains"},
		{"Oatmeal", 0.71, "grains"},
		{"Tortilla", 2.18, "grains"},
		{"Apple", 0.52, "fruit"},
		{"Banana", 0.89, "fruit"},
		{"Strawberries", 0.32, "fruit"},
		{"Orange", 0.47, "fruit"},
		{"Grapes", 0.69, "fruit"},
		{"Blueberries", 0.57, "fruit"},
		{"Broccoli", 0.34, "vegetable"},
		{"Spinach", 0.23, "vegetable"},
		{"Carrots", 0.41, "vegetable"},
		{"Sweet Potato", 0.86, "vegetable"},
		{"Corn", 0.96, "vegetable"},
		{"Whole Milk", 0.61, "dairy"},
		{"Cheddar Cheese", 4.03, "dairy"},
		{"Greek Yogurt", 0.59, "dairy"},
		{"Olive Oil", 8.84, "fats"},
		{"Peanut Butter", 5.88, "fats"},
		{"Almonds", 5.79, "fats"},
		{"Butter", 7.17, "fats"},
		{"Avocado", 1.60, "fats"},
	}

	for _, s := range seeds {
		foodID, err := dbNextFoodID()
		if err != nil {
			return err
		}
		err = dbInsertFood(Food{
			FoodID:   foodID,
			UserID:   userID,
			Name:     s.name,
			KcalPerG: s.kcal,
			FoodType: s.foodType,
		})
		if err != nil {
			return err
		}
	}
	return nil
}