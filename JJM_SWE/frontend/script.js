const API_BASE = "/api";

let selectedDate = getTodayDate();

function getTodayDate() {
  return new Date().toISOString().split("T")[0];
}

function buildDateTabs() {
  const tabs = document.getElementById("date-tabs");
  const wrapper = document.getElementById("date-tabs-wrapper");
  const today = new Date();
  const days = ["Sun","Mon","Tue","Wed","Thu","Fri","Sat"];
  const months = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];

  tabs.innerHTML = "";

  // 90 days back through 7 days forward
  for (let i = 90; i >= -7; i--) {
    const d = new Date(today);
    d.setDate(today.getDate() - i);
    const iso = d.toISOString().split("T")[0];

    const wrapper = document.createElement("div");
    wrapper.className = "date-tab-wrapper" + (iso === selectedDate ? " active" : "");
    wrapper.dataset.date = iso;

    const tab = document.createElement("div");
    tab.className = "date-tab";
    tab.innerHTML = `
      <span class="tab-day">${days[d.getDay()]}</span>
      <span class="tab-date">${months[d.getMonth()]} ${d.getDate()}</span>
    `;

    wrapper.appendChild(tab);

    wrapper.addEventListener("click", async () => {
      selectedDate = iso;
      document.querySelectorAll(".date-tab-wrapper").forEach(t => t.classList.remove("active"));
      wrapper.classList.add("active");
      await loadEntries();
      await loadGraph();
    });

    tabs.appendChild(wrapper);
  }

  // center today's tab in the scroll container
  const activeTab = tabs.querySelector(".date-tab-wrapper.active");
  if (activeTab) {
    setTimeout(() => {
      const tabLeft = activeTab.offsetLeft;
      const tabWidth = activeTab.offsetWidth;
      const containerWidth = wrapper.offsetWidth;
      tabs.scrollLeft = tabLeft - containerWidth / 2 + tabWidth / 2;
    }, 50);
  }
}

document.getElementById("scroll-left").addEventListener("click", () => {
  document.getElementById("date-tabs").scrollBy({ left: -160, behavior: "smooth" });
});
document.getElementById("scroll-right").addEventListener("click", () => {
  document.getElementById("date-tabs").scrollBy({ left: 160, behavior: "smooth" });
});

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
const usernameDisplay = document.getElementById("username-display");
const svg = d3.select("#graph");

// check session and redirect if not logged in
async function checkAuth() {
  try {
    const res = await fetch(`${API_BASE}/me`);
    if (!res.ok) {
      window.location.href = "/login.html";
      return false;
    }
    const data = await res.json();
    if (usernameDisplay && data.username) {
      usernameDisplay.textContent = data.username;
    }
    return true;
  } catch {
    window.location.href = "/login.html";
    return false;
  }
}

async function apiGet(url) {
  const res = await fetch(url);
  if (res.status === 401) {
    window.location.href = "/login.html";
    return;
  }
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

  if (res.status === 401) {
    window.location.href = "/login.html";
    return;
  }

  if (!res.ok) {
    throw new Error(await res.text());
  }

  return res.json();
}

async function apiDelete(url) {
  const res = await fetch(url, { method: "DELETE" });
  if (res.status === 401) {
    window.location.href = "/login.html";
    return;
  }
  if (!res.ok) {
    throw new Error(await res.text());
  }
  return res.json();
}

async function logout() {
  await fetch(`${API_BASE}/logout`, { method: "POST" });
  window.location.href = "/login.html";
}

// load and render food data
async function loadFoods() {
  const foods = await apiGet(`${API_BASE}/foods`);
  if (!foods) return;

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
  if (!foodsList) return;
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
  const date = selectedDate;
  const data = await apiGet(`${API_BASE}/entries?date=${encodeURIComponent(date)}`);
  if (!data) return;

  dailyTotal.textContent = `${data.daily_total_calories.toFixed(2)} cal`;
  entriesList.innerHTML = "";

  if (data.entries.length === 0) {
    entriesList.innerHTML = `<div class="small">No foods logged for this date.</div>`;
    return;
  }

  const PREVIEW = 3;
  const entries = [...data.entries].reverse();

  entriesList.innerHTML = "";

  entries.forEach((entry, i) => {
    const card = document.createElement("div");
    card.className = "entry-card";
    if (i >= PREVIEW) card.style.display = "none";

    const groupAccent = colorForGroup(entry.food_type);
    card.style.borderLeft = `4px solid ${groupAccent}`;

    const calColor = ["protein"].includes(entry.food_type)
      ? "var(--accent-protein)"
      : ["fruit","vegetable"].includes(entry.food_type)
        ? "var(--accent-green)"
        : "var(--accent-cal)";

    card.innerHTML = `
      <div class="row">
        <div>
          <div><strong>${entry.name}</strong></div>
          <div class="small">${entry.grams}g • ${entry.food_type} • <span style="color:${calColor};font-weight:600">${entry.calories} cal</span></div>
        </div>
        <button class="delete-btn" data-entry-id="${entry.entry_id}">Delete</button>
      </div>
    `;

    card.querySelector(".delete-btn").addEventListener("click", async () => {
      try {
        await apiDelete(`${API_BASE}/entry/${entry.entry_id}`);
        await refreshAll();
      } catch (err) {
        alert(String(err));
      }
    });

    entriesList.appendChild(card);
  });

  if (entries.length > PREVIEW) {
    const hidden = entries.length - PREVIEW;
    const toggle = document.createElement("button");
    toggle.className = "btn-toggle-log";
    toggle.textContent = `Show ${hidden} more ▼`;
    toggle.addEventListener("click", () => {
      const expanding = toggle.textContent.includes("▼");
      entriesList.querySelectorAll(".entry-card").forEach((c, i) => {
        if (i >= PREVIEW) c.style.display = expanding ? "block" : "none";
      });
      toggle.textContent = expanding ? "Show less ▲" : `Show ${hidden} more ▼`;
    });
    entriesList.appendChild(toggle);
  }
}

// graph rendering with d3
const GROUP_COLORS = {
  fruit:     "#ffd54f",
  vegetable: "#81c784",
  protein:   "#ef9a9a",
  grains:    "#d7ccc8",
  dairy:     "#90caf9",
  fats:      "#ffcc80",
  beverages: "#80deea",
  sweets:    "#ce93d8",
  other:     "#b0bec5",
};

function colorForGroup(group) {
  return GROUP_COLORS[group] || "#b0bec5";
}

// lighter tint for stroke (blend toward white)
function strokeForGroup(group) {
  const hex = colorForGroup(group).slice(1);
  const r = Math.min(255, parseInt(hex.slice(0,2),16) + 40);
  const g = Math.min(255, parseInt(hex.slice(2,4),16) + 40);
  const b = Math.min(255, parseInt(hex.slice(4,6),16) + 40);
  return `rgb(${r},${g},${b})`;
}

async function loadGraph() {
  const date = selectedDate;
  const data = await apiGet(`${API_BASE}/graph?date=${encodeURIComponent(date)}`);
  if (!data) return;

  graphTotal.textContent = `${data.daily_total_calories.toFixed(2)} cal`;
  renderGraph(data);
}

function renderGraph(data) {
  svg.selectAll("*").remove();

  const graphEl = document.getElementById("graph");
  const width = graphEl.clientWidth || 900;
  const height = graphEl.clientHeight || 700;

  svg.attr("viewBox", `0 0 ${width} ${height}`);

  // define radial gradients for each food group
  const defs = svg.append("defs");
  Object.entries(GROUP_COLORS).forEach(([group, baseColor]) => {
    const grad = defs.append("radialGradient")
      .attr("id", `grad-${group}`)
      .attr("cx", "40%").attr("cy", "35%")
      .attr("r", "60%");
    grad.append("stop").attr("offset", "0%")
      .attr("stop-color", strokeForGroup(group)).attr("stop-opacity", 1);
    grad.append("stop").attr("offset", "100%")
      .attr("stop-color", baseColor).attr("stop-opacity", 1);
  });

  const nodes = data.nodes.map((d) => ({ ...d }));
  const links = data.links.map((d) => ({ ...d }));

  const simulation = d3
    .forceSimulation(nodes)
    .force("link", d3.forceLink(links).id((d) => d.id).distance(120))
    .force("charge", d3.forceManyBody().strength(-500))
    .force("center", d3.forceCenter(width / 2, height / 2))
    .force("collision", d3.forceCollide().radius((d) => {
      const density = d.density || 0;
      return d.type === "group"
        ? Math.max(45, density * 12 + 10)
        : Math.max(22, density * 7 + 8);
    }));

  const link = svg
    .append("g")
    .selectAll("line")
    .data(links)
    .enter()
    .append("line")
    .attr("stroke-opacity", 0.6)
    .attr("stroke-width", 1.5)
    .attr("stroke-dasharray", "5,4");

  const node = svg
    .append("g")
    .selectAll("g")
    .data(nodes)
    .enter()
    .append("g");

  node
    .append("circle")
    .attr("r", (d) => {
      const density = d.density || 0;
      return d.type === "group"
        ? Math.max(35, density * 12)
        : Math.max(14, density * 7);
    })
    .attr("fill", (d) => `url(#grad-${d.group})`)
    .attr("stroke", (d) => strokeForGroup(d.group))
    .attr("stroke-width", (d) => d.type === "group" ? 2.5 : 2)
    .attr("stroke-opacity", 0.85);

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
      .attr("stroke", (d) => colorForGroup(d.source.group || d.source.id))
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
  const date = selectedDate;

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


document.getElementById("logout-btn").addEventListener("click", logout);

// verify auth then load everything
checkAuth().then((ok) => {
  if (ok) {
    buildDateTabs();
    refreshAll();
  }
});
