#!/usr/bin/env python3

import csv
import json
import os

BASE_DIR = os.path.dirname(__file__)
INPUT_CSV = os.path.join(BASE_DIR, "..", "data", "foods_seed.csv")
OUTPUT_JSON = os.path.join(BASE_DIR, "..", "data", "foods.json")


def main():
    if not os.path.exists(INPUT_CSV):
        print(f"No seed CSV found at {INPUT_CSV}")
        return

    foods = []
    with open(INPUT_CSV, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for i, row in enumerate(reader, start=1):
            calories = float(row["calories"])
            serving_grams = float(row["serving_grams"])
            kcal_per_g = round(calories / serving_grams, 2)

            foods.append({
                "food_id": f"food-{i}",
                "name": row["name"].strip(),
                "kcal_per_g": kcal_per_g,
                "food_type": row["food_type"].strip().lower()
            })

    with open(OUTPUT_JSON, "w", encoding="utf-8") as f:
        json.dump(foods, f, indent=2)

    print(f"Exported {len(foods)} foods to {OUTPUT_JSON}")


if __name__ == "__main__":
    main()