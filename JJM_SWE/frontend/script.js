const API_BASE = "http://localhost:8080/api";

const dateInput = document.getElementById("date-input");
const foodNameInput = document.getElementById("food-name");
const foodCaloriesInput = document.getElementById("food-calories");
const foodServingGramsInput = document.getElementById("food-serving-grams");
const foodTypeSelect = document.getElementById("food-type");
const createFoodBtn = document.getElementById("create-food-btn");
const createFoodMessage = document.getElementById("create-food-message");

const foodSelect = document.getElementById("food-select");
const entryGramsInput = document.getElementById("entry-grams");
const addEntryBtn = document.getElementById("add-entry-btn");
const addEntryMessage = document.getElementById("add-entry-message");

const foodsList = document.getElementById("foods-list");
const entriesList = document.getElementById("entries-list");
const dailyTotal = document.getElementById("daily-total");
const graphTotal = document.getElementById("graph-total");
const svg = d3.select("#graph");

function getTodayDate() {
  return new Date().toISOString().split("T")[0];
}

dateInput.value = getTodayDate();

async function apiGet(url) {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(await res.text());
  }
  return res.json();
}

async function apiPost(url, body) {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    throw new Error(await res.text());
  }

  return res.json();
}

async function apiDelete(url) {
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) {
    throw new Error(await res.text());
  }
  return res.json();
}

async function loadFoods() {
  const foods = await apiGet(`${API_BASE}/foods`);

  foodSelect.innerHTML = "";

  if (foods.length === 0) {
    const option = document.createElement("option");
    option.value = "";
    option.textContent = "No foods saved yet";
    foodSelect.appendChild(option);
  } else {
    for (const food of foods) {
      const option = document.createElement("option");
      option.value = food.food_id;
      option.textContent = `${food.name} (${food.food_type}, ${food.kcal_per_g} kcal/g)`;
      foodSelect.appendChild(option);
    }
  }

  renderFoodsList(foods);
}

function renderFoodsList(foods) {
  foodsList.innerHTML = "";

  if (foods.length === 0) {
    foodsList.innerHTML = `<div class="small">No saved foods yet.</div>`;
    return;
  }

  for (const food of foods) {
    const card = document.createElement("div");
    card.className = "food-card";

    card.innerHTML = `
      <div class="row">
        <div>
          <div><strong>${food.name}</strong></div>
          <div class="small">${food.food_type} • ${food.kcal_per_g} kcal/g</div>
        </div>
        <button class="delete-btn" data-food-id="${food.food_id}">Delete</button>
      </div>
    `;

    foodsList.appendChild(card);
  }

  document.querySelectorAll("[data-food-id]").forEach((btn) => {
    btn.addEventListener("click", async (e) => {
      const target = e.currentTarget;
      const foodID = target.dataset.foodId;
      if (!foodID) return;

      try {
        await apiDelete(`${API_BASE}/foods/${foodID}`);
        await refreshAll();
      } catch (err) {
        alert(String(err));
      }
    });
  });
}

async function loadEntries() {
  const date = dateInput.value;
  const data = await apiGet(`${API_BASE}/entries?date=${encodeURIComponent(date)}`);

  dailyTotal.textContent = `${data.daily_total_calories.toFixed(2)} cal`;
  entriesList.innerHTML = "";

  if (data.entries.length === 0) {
    entriesList.innerHTML = `<div class="small">No foods logged for this date.</div>`;
    return;
  }

  for (const entry of data.entries) {
    const card = document.createElement("div");
    card.className = "entry-card";

    card.innerHTML = `
      <div class="row">
        <div>
          <div><strong>${entry.name}</strong></div>
          <div class="small">${entry.grams}g • ${entry.food_type} • ${entry.calories} cal</div>
        </div>
        <button class="delete-btn" data-entry-id="${entry.entry_id}">Delete</button>
      </div>
    `;

    entriesList.appendChild(card);
  }

  document.querySelectorAll("[data-entry-id]").forEach((btn) => {
    btn.addEventListener("click", async (e) => {
      const target = e.currentTarget;
      const entryID = target.dataset.entryId;
      if (!entryID) return;

      try {
        await apiDelete(`${API_BASE}/entry/${entryID}`);
        await refreshAll();
      } catch (err) {
        alert(String(err));
      }
    });
  });
}

function colorForGroup(group) {
  const colors = {
    fruit: "#ffd54f",
    vegetable: "#81c784",
    protein: "#ef9a9a",
    grains: "#d7ccc8",
    dairy: "#90caf9",
    fats: "#ffcc80",
    beverages: "#80deea",
    sweets: "#ce93d8",
    other: "#b0bec5",
  };
  return colors[group] || "#b0bec5";
}

async function loadGraph() {
  const date = dateInput.value;
  const data = await apiGet(`${API_BASE}/graph?date=${encodeURIComponent(date)}`);
  graphTotal.textContent = `${data.daily_total_calories.toFixed(2)} cal`;
  renderGraph(data);
}

function renderGraph(data) {
  svg.selectAll("*").remove();

  const graphEl = document.getElementById("graph");
  const width = graphEl.clientWidth || 900;
  const height = graphEl.clientHeight || 700;

  svg.attr("viewBox", `0 0 ${width} ${height}`);

  const nodes = data.nodes.map((d) => ({ ...d }));
  const links = data.links.map((d) => ({ ...d }));

  const simulation = d3
    .forceSimulation(nodes)
    .force("link", d3.forceLink(links).id((d) => d.id).distance(120))
    .force("charge", d3.forceManyBody().strength(-500))
    .force("center", d3.forceCenter(width / 2, height / 2))
    .force("collision", d3.forceCollide().radius((d) => {
      return d.type === "group"
        ? Math.max(45, d.value * 0.15)
        : Math.max(22, d.value * 0.08);
    }));

  const link = svg
    .append("g")
    .selectAll("line")
    .data(links)
    .enter()
    .append("line")
    .attr("stroke", "#fff")
    .attr("stroke-opacity", 0.8)
    .attr("stroke-width", 1.5);

  const node = svg
    .append("g")
    .selectAll("g")
    .data(nodes)
    .enter()
    .append("g");

  node
    .append("circle")
    .attr("r", (d) =>
      d.type === "group"
        ? Math.max(35, d.value * 0.12)
        : Math.max(14, d.value * 0.06)
    )
    .attr("fill", (d) => colorForGroup(d.group));

  node
    .append("text")
    .attr("text-anchor", "middle")
    .attr("fill", "#111")
    .style("font-size", (d) => (d.type === "group" ? "12px" : "10px"))
    .each(function (d) {
      const lines = d.label.split("\n");
      const text = d3.select(this);
      text.text(null);
      lines.forEach((line, i) => {
        text
          .append("tspan")
          .text(line)
          .attr("x", 0)
          .attr("dy", i === 0 ? 0 : 12);
      });
    });

  node.call(
    d3.drag()
      .on("start", (event, d) => {
        if (!event.active) simulation.alphaTarget(0.3).restart();
        d.fx = d.x;
        d.fy = d.y;
      })
      .on("drag", (event, d) => {
        d.fx = event.x;
        d.fy = event.y;
      })
      .on("end", (event, d) => {
        if (!event.active) simulation.alphaTarget(0);
        d.fx = null;
        d.fy = null;
      })
  );

  simulation.on("tick", () => {
    link
      .attr("x1", (d) => d.source.x)
      .attr("y1", (d) => d.source.y)
      .attr("x2", (d) => d.target.x)
      .attr("y2", (d) => d.target.y);

    node.attr("transform", (d) => `translate(${d.x},${d.y})`);
  });
}

async function refreshAll() {
  await loadFoods();
  await loadEntries();
  await loadGraph();
}

createFoodBtn.addEventListener("click", async () => {
  createFoodMessage.textContent = "";

  const name = foodNameInput.value.trim();
  const calories = Number(foodCaloriesInput.value);
  const servingGrams = Number(foodServingGramsInput.value);
  const foodType = foodTypeSelect.value;

  if (!name || calories <= 0 || servingGrams <= 0 || !foodType) {
    createFoodMessage.textContent = "Enter valid food fields.";
    return;
  }

  try {
    await apiPost(`${API_BASE}/foods`, {
      name,
      calories,
      serving_grams: servingGrams,
      food_type: foodType,
    });

    foodNameInput.value = "";
    foodCaloriesInput.value = "";
    foodServingGramsInput.value = "";
    createFoodMessage.textContent = "Food saved.";

    await refreshAll();
  } catch (err) {
    console.error("Create food failed:", err);
    createFoodMessage.textContent = `Could not save food: ${err.message || err}`;
  }
});

addEntryBtn.addEventListener("click", async () => {
  addEntryMessage.textContent = "";

  const foodID = foodSelect.value;
  const grams = Number(entryGramsInput.value);
  const date = dateInput.value;

  if (!foodID || grams <= 0) {
    addEntryMessage.textContent = "Select a food and enter valid grams.";
    return;
  }

  try {
    await apiPost(`${API_BASE}/entry`, {
      food_id: foodID,
      grams,
      date,
    });

    entryGramsInput.value = "";
    addEntryMessage.textContent = "Entry added.";

    await refreshAll();
  } catch (err) {
    addEntryMessage.textContent = String(err);
  }
});

dateInput.addEventListener("change", async () => {
  await loadEntries();
  await loadGraph();
});

refreshAll();
 