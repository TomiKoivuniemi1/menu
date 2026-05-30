const euro = cents => (cents / 100).toFixed(2) + " €";

const allowedCategories = ["Meal", "Lunch", "Drinks"];

let menu = [];
let orders = [];

const $ = id => document.getElementById(id);

async function api(path, options = {}) {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options
  });
  if (!res.ok) {
    const msg = await res.text();
    throw new Error(msg);
  }
  return res.json();
}

async function loadAll() {
  menu = await api("/api/menu");
  orders = await api("/api/orders");

  if (!Array.isArray(menu)) menu = [];
  if (!Array.isArray(orders)) orders = [];

  renderCategorySelect();
  renderMenu();
  renderOrders();
  renderCategoryFilter();
  updateSerialStatus();
}

function renderOrders() {
  const root = $("orders");
  if (!orders.length) {
    root.innerHTML = `<p class="empty">No orders yet.</p>`;
    return;
  }

  root.innerHTML = orders.map(order => `
    <article class="order ${order.status}">
      <div class="order-head">
        <strong>#${order.id} ${escapeHtml(order.table || "")}</strong>
        <span>${order.status}</span>
      </div>
      <ul>
        ${(Array.isArray(order.lines) ? order.lines : []).map(line => `
          <li>${line.quantity}x ${escapeHtml(line.name)} <span>${euro(line.priceCents * line.quantity)}</span></li>
        `).join("")}
      </ul>
      <div class="order-total">Total: ${euro(order.totalCents)}</div>
      <div class="actions">
        <button onclick="setOrderStatus(${order.id}, 'accepted')" ${order.status !== "received" ? "disabled" : ""}>Accept</button>
        <button onclick="setOrderStatus(${order.id}, 'completed')" ${order.status === "completed" ? "disabled" : ""}>Complete</button>
      </div>
    </article>
  `).join("");
}

function renderMenu() {
  const search = $("search").value.toLowerCase();
  const category = $("categoryFilter").value;

  const filtered = menu.filter(item => {
    const matchesSearch =
      item.name.toLowerCase().includes(search) ||
      item.category.toLowerCase().includes(search);

    const matchesCategory = !category || item.category === category;

    return matchesSearch && matchesCategory;
  });

  $("menuItems").innerHTML = filtered.map(item => `
    <article class="menu-row">
      <div>
        <strong>${escapeHtml(item.name)}</strong>
        <span>
          ${escapeHtml(item.category)} - ${euro(item.priceCents)}
        </span>
      </div>
      <div class="actions">
        <button onclick="editItem(${item.id})">Edit</button>
        <button onclick="deleteItem(${item.id})">Delete</button>
      </div>
    </article>
  `).join("");
}

function renderCategorySelect() {
  $("category").innerHTML =
    `<option value="">Choose category</option>` +
    allowedCategories.map(cat =>
      `<option value="${escapeAttr(cat)}">${escapeHtml(cat)}</option>`
    ).join("");
}

function renderCategoryFilter() {
  const selected = $("categoryFilter").value;

  $("categoryFilter").innerHTML =
    `<option value="">All categories</option>` +
    allowedCategories.map(cat =>
      `<option value="${escapeAttr(cat)}" ${cat === selected ? "selected" : ""}>${escapeHtml(cat)}</option>`
    ).join("");
}

async function setOrderStatus(id, status) {
  const updated = await api(`/api/orders/${id}/status`, {
    method: "PUT",
    body: JSON.stringify({ status })
  });

  const idx = orders.findIndex(o => o.id === id);
  if (idx >= 0) orders[idx] = updated;

  renderOrders();
}

async function clearAllOrders() {
  if (!confirm("Clear all orders?")) return;

  await api("/api/orders/clear", {
    method: "POST"
  });

  orders = [];
  renderOrders();
}

function editItem(id) {
  const item = menu.find(i => i.id === id);
  if (!item) return;

  $("itemId").value = item.id;
  $("category").value = allowedCategories.includes(item.category) ? item.category : "";
  $("name").value = item.name;
  $("price").value = (item.priceCents / 100).toFixed(2);

  window.scrollTo({ top: 0, behavior: "smooth" });
}

async function deleteItem(id) {
  if (!confirm("Delete this item?")) return;

  await api(`/api/menu/${id}`, { method: "DELETE" });

  menu = menu.filter(i => i.id !== id);
  renderMenu();
  renderCategoryFilter();
}

function resetForm() {
  $("itemId").value = "";
  $("category").value = "";
  $("name").value = "";
  $("price").value = "";
}

$("itemForm").addEventListener("submit", async e => {
  e.preventDefault();

  const id = $("itemId").value;
  const category = $("category").value;

  if (!allowedCategories.includes(category)) {
    alert("Choose one of the LCD-supported categories.");
    return;
  }

  const payload = {
    category,
    name: $("name").value.trim(),
    priceCents: Math.round(parseFloat($("price").value.replace(",", ".")) * 100),
    available: true
  };

  if (id) {
    const updated = await api(`/api/menu/${id}`, {
      method: "PUT",
      body: JSON.stringify(payload)
    });

    const idx = menu.findIndex(i => i.id === updated.id);
    if (idx >= 0) menu[idx] = updated;
  } else {
    const created = await api("/api/menu", {
      method: "POST",
      body: JSON.stringify(payload)
    });

    menu.push(created);
  }

  resetForm();
  renderMenu();
  renderCategoryFilter();
});

$("resetFormBtn").addEventListener("click", resetForm);
$("search").addEventListener("input", renderMenu);
$("categoryFilter").addEventListener("change", renderMenu);
$("clearOrdersBtn").addEventListener("click", clearAllOrders);

$("sendMenuBtn").addEventListener("click", async () => {
  await api("/api/serial/send-menu", { method: "POST" });
  alert("Menu update sent to serial device, if connected.");
});

async function updateSerialStatus() {
  try {
    const st = await api("/api/serial/status");

    $("serialStatus").textContent =
      `Serial: ${
        st.enabled
          ? (st.connected
              ? (st.deviceOnline ? "device online" : "connected")
              : "not connected")
          : "disabled"
      } | ${st.port} @ ${st.baud}`;

    $("serialStatus").className = "status " + (st.connected ? "ok" : "warn");
  } catch {
    $("serialStatus").textContent = "Serial: unknown";
  }
}

function connectEvents() {
  const events = new EventSource("/events");

  events.onmessage = ev => {
    const msg = JSON.parse(ev.data);

    if (msg.type === "order_created") {
      if (!orders.some(o => o.id === msg.data.id)) orders.unshift(msg.data);
      renderOrders();
    }

    if (msg.type === "order_updated") {
      const idx = orders.findIndex(o => o.id === msg.data.id);
      if (idx >= 0) orders[idx] = msg.data;
      renderOrders();
    }

    if (msg.type === "orders_cleared") {
      orders = [];
      renderOrders();
    }

    if (msg.type === "menu_changed") {
      loadAll();
    }
  };
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, c => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#039;"
  }[c]));
}

function escapeAttr(s) {
  return escapeHtml(s).replace(/"/g, "&quot;");
}

setInterval(updateSerialStatus, 3000);
connectEvents();
loadAll();