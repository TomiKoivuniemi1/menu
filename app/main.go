package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

type MenuItem struct {
	ID         int    `json:"id"`
	Category   string `json:"category"`
	Name       string `json:"name"`
	PriceCents int    `json:"priceCents"`
	Available  bool   `json:"available"`
}

type OrderLine struct {
	ItemID     int    `json:"itemId"`
	Name       string `json:"name"`
	Quantity   int    `json:"quantity"`
	PriceCents int    `json:"priceCents"`
}

type Order struct {
	ID         int         `json:"id"`
	DeviceID   string      `json:"deviceId"`
	Table      string      `json:"table"`
	Status     string      `json:"status"`
	Lines      []OrderLine `json:"lines"`
	TotalCents int         `json:"totalCents"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

type SerialStatus struct {
	Enabled        bool   `json:"enabled"`
	Connected      bool   `json:"connected"`
	DeviceOnline   bool   `json:"deviceOnline"`
	Port           string `json:"port"`
	Baud           int    `json:"baud"`
	LastRX         string `json:"lastRx"`
	LastTX         string `json:"lastTx"`
	LastError      string `json:"lastError"`
	LastDeviceSeen string `json:"lastDeviceSeen"`
	LastUpdate     string `json:"lastUpdate"`
}

type Store struct {
	mu          sync.RWMutex
	menu        []MenuItem
	orders      []Order
	nextItemID  int
	nextOrderID int
	menuPath    string
	ordersPath  string
	broadcaster *Broadcaster
	serial      *SerialManager
}

func NewStore(menuPath, ordersPath string) *Store {
	s := &Store{menuPath: menuPath, ordersPath: ordersPath, nextItemID: 1, nextOrderID: 1, broadcaster: NewBroadcaster()}
	if err := s.LoadMenu(); err != nil {
		log.Printf("menu load failed, using defaults: %v", err)
		s.menu = defaultMenu()
		s.recalculateItemIDs()
		_ = s.SaveMenu()
	}
	if err := s.LoadOrders(); err != nil {
		log.Printf("orders load failed, starting empty: %v", err)
		s.orders = []Order{}
		s.recalculateOrderIDs()
		_ = s.SaveOrders()
	}
	return s
}

func validCategory(category string) bool {
	switch category {
	case "Meal", "Lunch", "Drinks":
		return true
	default:
		return false
	}
}

func (s *Store) recalculateItemIDs() {
	maxID := 0
	for _, item := range s.menu {
		if item.ID > maxID {
			maxID = item.ID
		}
	}
	s.nextItemID = maxID + 1
}

func (s *Store) recalculateOrderIDs() {
	maxID := 0
	for _, order := range s.orders {
		if order.ID > maxID {
			maxID = order.ID
		}
	}
	s.nextOrderID = maxID + 1
}

func (s *Store) LoadMenu() error {
	b, err := os.ReadFile(s.menuPath)
	if err != nil {
		return err
	}
	var items []MenuItem
	if err := json.Unmarshal(b, &items); err != nil {
		return err
	}
	s.menu = items
	s.recalculateItemIDs()
	return nil
}

func (s *Store) SaveMenu() error {
	if err := os.MkdirAll(filepath.Dir(s.menuPath), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.menu, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.menuPath, b, 0644)
}

func (s *Store) LoadOrders() error {
	b, err := os.ReadFile(s.ordersPath)
	if err != nil {
		return err
	}
	var orders []Order
	if err := json.Unmarshal(b, &orders); err != nil {
		return err
	}
	s.orders = orders
	s.recalculateOrderIDs()
	return nil
}

func (s *Store) SaveOrders() error {
	if err := os.MkdirAll(filepath.Dir(s.ordersPath), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.orders, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.ordersPath, b, 0644)
}

func (s *Store) ListMenu() []MenuItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]MenuItem(nil), s.menu...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category == out[j].Category {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		}

		return strings.ToLower(out[i].Category) < strings.ToLower(out[j].Category)
	})
	return out
}

func (s *Store) AddMenuItem(item MenuItem) (MenuItem, error) {
	item.Name = strings.TrimSpace(item.Name)
	item.Category = strings.TrimSpace(item.Category)
	if item.Name == "" || item.Category == "" || item.PriceCents < 0 {
		return MenuItem{}, errors.New("name, category and valid price are required")
	}

	if !validCategory(item.Category) {
		return MenuItem{}, errors.New("invalid category")
	}
	s.mu.Lock()
	item.ID = s.nextItemID
	s.nextItemID++
	s.menu = append(s.menu, item)
	err := s.SaveMenu()
	s.mu.Unlock()
	if err == nil {
		s.broadcast("menu_changed", item)
		go s.sendMenuUpdateToDevice()
	}
	return item, err
}

func (s *Store) UpdateMenuItem(id int, item MenuItem) (MenuItem, error) {
	item.Name = strings.TrimSpace(item.Name)
	item.Category = strings.TrimSpace(item.Category)
	if item.Name == "" || item.Category == "" || item.PriceCents < 0 {
		return MenuItem{}, errors.New("name, category and valid price are required")
	}

	if !validCategory(item.Category) {
		return MenuItem{}, errors.New("invalid category")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.menu {
		if s.menu[i].ID == id {
			item.ID = id
			s.menu[i] = item
			err := s.SaveMenu()
			if err == nil {
				s.broadcast("menu_changed", item)
				go s.sendMenuUpdateToDevice()
			}
			return item, err
		}
	}
	return MenuItem{}, http.ErrMissingFile
}

func (s *Store) DeleteMenuItem(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.menu {
		if s.menu[i].ID == id {
			s.menu = append(s.menu[:i], s.menu[i+1:]...)
			err := s.SaveMenu()
			if err == nil {
				s.broadcast("menu_changed", map[string]int{"id": id})
				go s.sendMenuUpdateToDevice()
			}
			return err
		}
	}
	return http.ErrMissingFile
}

func (s *Store) CreateOrderFromDevice(deviceID, table string, pairs map[int]int) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	itemByID := map[int]MenuItem{}
	for _, item := range s.menu {
		itemByID[item.ID] = item
	}
	var lines []OrderLine
	total := 0
	for itemID, qty := range pairs {
		if qty <= 0 {
			continue
		}
		item, ok := itemByID[itemID]
		if !ok || !item.Available {
			continue
		}
		lines = append(lines, OrderLine{ItemID: item.ID, Name: item.Name, Quantity: qty, PriceCents: item.PriceCents})
		total += item.PriceCents * qty
	}
	if len(lines) == 0 {
		return Order{}, errors.New("order contains no valid available items")
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].ItemID < lines[j].ItemID })
	now := time.Now()
	order := Order{ID: s.nextOrderID, DeviceID: deviceID, Table: table, Status: "received", Lines: lines, TotalCents: total, CreatedAt: now, UpdatedAt: now}
	s.nextOrderID++
	s.orders = append([]Order{order}, s.orders...)
	if err := s.SaveOrders(); err != nil {
		return Order{}, err
	}
	s.broadcast("order_created", order)
	return order, nil
}

func (s *Store) ListOrders() []Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Order(nil), s.orders...)
}

func (s *Store) UpdateOrderStatus(id int, status string) (Order, error) {
	if status != "received" && status != "accepted" && status != "completed" {
		return Order{}, errors.New("invalid status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.orders {
		if s.orders[i].ID == id {
			s.orders[i].Status = status
			s.orders[i].UpdatedAt = time.Now()
			order := s.orders[i]
			if err := s.SaveOrders(); err != nil {
				return Order{}, err
			}
			s.broadcast("order_updated", order)
			if s.serial != nil {
				switch status {
				case "accepted":
					s.serial.Send(fmt.Sprintf("A%d\n", order.ID))
				case "completed":
					s.serial.Send(fmt.Sprintf("R%d\n", order.ID))
				}
			}
			return order, nil
		}
	}
	return Order{}, http.ErrMissingFile
}

func (s *Store) ClearOrders() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders = []Order{}
	s.nextOrderID = 1
	if err := s.SaveOrders(); err != nil {
		return err
	}
	s.broadcast("orders_cleared", map[string]string{"status": "cleared"})
	if s.serial != nil {
		s.serial.Send("C\n")
	}
	return nil
}

func (s *Store) SendFullMenuToDevice() error {
	items := s.ListMenu()
	if s.serial == nil {
		return errors.New("serial manager not configured")
	}
	s.serial.Send("MENU_BEGIN\n")
	for _, item := range items {
		available := 0
		if item.Available {
			available = 1
		}
		s.serial.Send(fmt.Sprintf("MENU_ITEM|%d|%s|%s|%d|%d\n", item.ID, sanitizeProtocolField(item.Category), sanitizeProtocolField(item.Name), item.PriceCents, available))
	}
	s.serial.Send("MENU_END\n")
	s.serial.Send("MENU_CHANGED\n")
	s.broadcast("menu_sent", map[string]int{"count": len(items)})
	return nil
}

func (s *Store) sendMenuUpdateToDevice() {
	if s.serial == nil || !s.serial.IsConnected() {
		return
	}
	if err := s.SendFullMenuToDevice(); err != nil {
		log.Printf("menu auto-send failed: %v", err)
	}
}

func (s *Store) broadcast(kind string, data any) {
	if s.broadcaster != nil {
		s.broadcaster.Send(kind, data)
	}
}

type Broadcaster struct {
	mu      sync.Mutex
	clients map[chan string]bool
}

func NewBroadcaster() *Broadcaster { return &Broadcaster{clients: map[chan string]bool{}} }
func (b *Broadcaster) Add() chan string {
	ch := make(chan string, 16)
	b.mu.Lock()
	b.clients[ch] = true
	b.mu.Unlock()
	return ch
}
func (b *Broadcaster) Remove(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}
func (b *Broadcaster) Send(kind string, data any) {
	payload, _ := json.Marshal(map[string]any{"type": kind, "data": data})
	msg := "data: " + string(payload) + "\n\n"
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

type SerialManager struct {
	mu       sync.RWMutex
	enabled  bool
	portName string
	baud     int
	port     serial.Port
	status   SerialStatus
	onLine   func(string)
}

func NewSerialManager(enabled bool, portName string, baud int, onLine func(string)) *SerialManager {
	return &SerialManager{enabled: enabled, portName: portName, baud: baud, onLine: onLine, status: SerialStatus{Enabled: enabled, Port: portName, Baud: baud}}
}
func (sm *SerialManager) Start() {
	if !sm.enabled {
		log.Println("serial disabled; run with -serial=true -port COMx")
		return
	}
	go sm.connectLoop()
}
func (sm *SerialManager) connectLoop() {
	for {
		sm.setError("")
		p, err := serial.Open(sm.portName, &serial.Mode{BaudRate: sm.baud})
		if err != nil {
			sm.setConnected(false)
			sm.setError(err.Error())
			time.Sleep(2 * time.Second)
			continue
		}
		sm.mu.Lock()
		sm.port = p
		sm.status.Connected = true
		sm.status.LastError = ""
		sm.status.LastUpdate = time.Now().Format(time.RFC3339)
		sm.mu.Unlock()
		log.Printf("serial connected on %s at %d", sm.portName, sm.baud)
		sm.Send("HELLO_SHOP\n")
		sm.readLoop(p)
		_ = p.Close()
		sm.mu.Lock()
		if sm.port == p {
			sm.port = nil
		}
		sm.status.Connected = false
		sm.status.DeviceOnline = false
		sm.status.LastUpdate = time.Now().Format(time.RFC3339)
		sm.mu.Unlock()
		time.Sleep(2 * time.Second)
	}
}
func (sm *SerialManager) readLoop(p serial.Port) {
	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		sm.mu.Lock()
		sm.status.LastRX = line
		sm.status.LastUpdate = time.Now().Format(time.RFC3339)
		if isDevicePresenceLine(line) {
			sm.status.DeviceOnline = true
			sm.status.LastDeviceSeen = time.Now().Format(time.RFC3339)
		}
		sm.mu.Unlock()
		if shouldLogSerialLine(line) {
			log.Printf("RX: %s", line)
		}
		if sm.onLine != nil {
			sm.onLine(line)
		}
	}
	if err := scanner.Err(); err != nil {
		sm.setError(err.Error())
	}
}
func (sm *SerialManager) Send(line string) {
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	sm.mu.RLock()
	p := sm.port
	sm.mu.RUnlock()
	if p == nil {
		sm.setError("serial not connected")
		return
	}
	_, err := p.Write([]byte(line))
	sm.mu.Lock()
	sm.status.LastTX = strings.TrimSpace(line)
	if err != nil {
		sm.status.LastError = err.Error()
	} else {
		sm.status.LastError = ""
	}
	sm.status.LastUpdate = time.Now().Format(time.RFC3339)
	sm.mu.Unlock()
	if shouldLogSerialLine(strings.TrimSpace(line)) {
		log.Printf("TX: %s", strings.TrimSpace(line))
	}
}
func (sm *SerialManager) Status() SerialStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	st := sm.status
	st.Enabled = sm.enabled
	st.Port = sm.portName
	st.Baud = sm.baud
	return st
}
func (sm *SerialManager) IsConnected() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.port != nil && sm.status.Connected
}
func (sm *SerialManager) setConnected(v bool) {
	sm.mu.Lock()
	sm.status.Connected = v
	sm.status.LastUpdate = time.Now().Format(time.RFC3339)
	sm.mu.Unlock()
}
func (sm *SerialManager) setError(msg string) {
	sm.mu.Lock()
	sm.status.LastError = msg
	sm.status.LastUpdate = time.Now().Format(time.RFC3339)
	sm.mu.Unlock()
}

func (sm *SerialManager) SendRepeat(line string) {
	for i := 0; i < 3; i++ {
		sm.Send(line)
		time.Sleep(120 * time.Millisecond)
	}
}

func handleSerialLine(store *Store, line string) {
	parts := strings.Split(line, "|")
	command := strings.TrimSpace(parts[0])
	switch {
	case command == "PING" || strings.HasPrefix(command, "PING "):
		if store.serial != nil {
			store.serial.Send("PONG\n")
		}
	case command == "MENU_DEVICE_BOOT" || command == "BT_TEST_READY" || command == "HELLO":
		if store.serial != nil {
			store.serial.Send("HELLO_SHOP\n")
		}
	case command == "ORDER":
		if len(parts) < 3 {
			log.Printf("invalid ORDER: %s", line)
			if store.serial != nil {
				store.serial.Send("ORDER_ERROR|BAD_FORMAT\n")
			}
			return
		}
		table := strings.TrimSpace(parts[1])
		if table == "" {
			table = "TABLE1"
		}
		deviceID := "customer-device"
		if len(parts) >= 4 && strings.TrimSpace(parts[3]) != "" {
			deviceID = strings.TrimSpace(parts[3])
		}
		pairs, err := parseItemPairs(parts[2])
		if err != nil {
			log.Printf("invalid ORDER items: %v", err)
			if store.serial != nil {
				store.serial.Send("ORDER_ERROR|BAD_ITEMS\n")
			}
			return
		}
		order, err := store.CreateOrderFromDevice(deviceID, table, pairs)
		if err != nil {
			log.Printf("order rejected: %v", err)
			if store.serial != nil {
				store.serial.Send("ORDER_ERROR|EMPTY_OR_UNAVAILABLE\n")
			}
			return
		}
		if store.serial != nil {
			store.serial.Send(fmt.Sprintf("V%d\n", order.ID))
		}
	default:
		log.Printf("unknown serial command: %s", line)
	}
}

func parseItemPairs(input string) (map[int]int, error) {
	out := map[int]int{}
	for _, pair := range strings.Split(input, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad item pair %q", pair)
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}
		qty, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		if id > 0 && qty > 0 {
			out[id] += qty
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no item pairs")
	}
	return out, nil
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	menuPath := flag.String("menu", "data/menu.json", "menu JSON file")
	ordersPath := flag.String("orders", "data/orders.json", "orders JSON file")

	serialEnabled := flag.Bool("serial", true, "enable serial USB/Bluetooth COM communication")
	port := flag.String("port", "COM5", "Bluetooth COM port")
	baud := flag.Int("baud", 9600, "serial baud rate")

	flag.Parse()

	store := NewStore(*menuPath, *ordersPath)
	serialManager := NewSerialManager(*serialEnabled, *port, *baud, func(line string) { handleSerialLine(store, line) })
	store.serial = serialManager
	serialManager.Start()

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("web")))
	mux.HandleFunc("/api/menu", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, store.ListMenu())
		case http.MethodPost:
			var item MenuItem
			if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			created, err := store.AddMenuItem(item)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, created)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/menu/", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/menu/"))
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		switch r.Method {
		case http.MethodPut:
			var item MenuItem
			if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			updated, err := store.UpdateMenuItem(id, item)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeJSON(w, updated)
		case http.MethodDelete:
			if err := store.DeleteMenuItem(id); err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeJSON(w, map[string]string{"status": "deleted"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, store.ListOrders())
		case http.MethodPost:
			var req struct {
				Table string         `json:"table"`
				Items map[string]int `json:"items"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			pairs := map[int]int{}
			for k, qty := range req.Items {
				id, _ := strconv.Atoi(k)
				pairs[id] = qty
			}
			order, err := store.CreateOrderFromDevice("browser-test", req.Table, pairs)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, order)
		case http.MethodDelete:
			if err := store.ClearOrders(); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, map[string]string{"status": "cleared"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/orders/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := store.ClearOrders(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, map[string]string{"status": "cleared"})
	})
	mux.HandleFunc("/api/orders/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/orders/")
		parts := strings.Split(path, "/")
		if len(parts) != 2 || parts[1] != "status" {
			writeError(w, http.StatusNotFound, errors.New("not found"))
			return
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		order, err := store.UpdateOrderStatus(id, req.Status)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, order)
	})
	mux.HandleFunc("/api/serial/status", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, store.serial.Status()) })
	mux.HandleFunc("/api/serial/send-menu", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := store.SendFullMenuToDevice(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, map[string]string{"status": "sent"})
	})
	mux.HandleFunc("/api/serial/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Line string `json:"line"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(req.Line) == "" {
			writeError(w, http.StatusBadRequest, errors.New("line is required"))
			return
		}
		store.serial.Send(req.Line)
		writeJSON(w, map[string]string{"status": "sent"})
	})
	mux.HandleFunc("/api/serial/test-order", func(w http.ResponseWriter, r *http.Request) {
		order, err := store.CreateOrderFromDevice("test-device", "TABLE1", map[int]int{1: 1, 48: 1})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, order)
	})
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		ch := store.broadcaster.Add()
		defer store.broadcaster.Remove(ch)
		fmt.Fprint(w, "data: {\"type\":\"connected\"}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		for {
			select {
			case <-r.Context().Done():
				return
			case msg := <-ch:
				fmt.Fprint(w, msg)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
	})
	log.Printf("Shop app running at http://localhost%s", *addr)
	log.Printf("serial enabled=%v port=%s baud=%d", *serialEnabled, *port, *baud)
	if err := http.ListenAndServe(*addr, logMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}
func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/serial/status" {
			log.Printf("%s %s", r.Method, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
func sanitizeProtocolField(s string) string {
	s = strings.ReplaceAll(s, "|", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}
func isDevicePresenceLine(line string) bool {
	return line == "PING" || strings.HasPrefix(line, "PING ") || line == "MENU_DEVICE_BOOT" || line == "BT_TEST_READY" || strings.HasPrefix(line, "HELLO")
}
func shouldLogSerialLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	if line == "PING" || strings.HasPrefix(line, "PING ") || line == "PONG" {
		return false
	}
	return true
}

func defaultMenu() []MenuItem {
	return []MenuItem{
		{1, "Hot Drinks", "The Exhibitor's Espresso", 280, true}, {2, "Hot Drinks", "Gallery Americano", 320, true}, {3, "Hot Drinks", "Curator's Cappuccino", 360, true}, {4, "Hot Drinks", "Artisan Latte", 360, true}, {5, "Hot Drinks", "Minimalist Flat White", 340, true}, {6, "Hot Drinks", "Masterpiece Mocha", 390, true}, {7, "Hot Drinks", "Velvet Exhibit Hot Chocolate", 390, true}, {8, "Hot Drinks", "Heritage Breakfast Tea", 280, true}, {9, "Hot Drinks", "Portrait of Earl Grey", 280, true}, {10, "Hot Drinks", "Zen Garden Green Tea", 320, true}, {11, "Hot Drinks", "Spice Gallery Chai Latte", 390, true}, {12, "Hot Drinks", "Botanical Infusion", 320, true},
		{13, "Cold Drinks", "Chilled Canvas Americano", 340, true}, {14, "Cold Drinks", "Sculptor's Iced Latte", 390, true}, {15, "Cold Drinks", "Frost & Cocoa Mocha", 420, true}, {16, "Cold Drinks", "Citrus Showcase Lemonade", 280, true}, {17, "Cold Drinks", "Crystal Collection Sparkling Water", 230, true}, {18, "Cold Drinks", "Pure Perspective Still Water", 200, true}, {19, "Cold Drinks", "Sunshine Gallery Orange Juice", 340, true}, {20, "Cold Drinks", "Orchard Exhibition Apple Juice", 340, true}, {21, "Cold Drinks", "Berry Composition Smoothie", 500, true}, {22, "Cold Drinks", "Tropical Installation Smoothie", 500, true}, {23, "Cold Drinks", "Refreshment Pavilion Iced Tea", 340, true}, {24, "Cold Drinks", "Pop Art Fizzy Drink", 280, true},
		{25, "Breakfast", "The French Exhibit Croissant", 280, true}, {26, "Breakfast", "Chocolate Masterpiece Pastry", 320, true}, {27, "Breakfast", "Renaissance Eggs Benedict", 950, true}, {28, "Breakfast", "Modern Avocado Canvas", 850, true}, {29, "Breakfast", "Grand Exhibition English Breakfast", 1350, true}, {30, "Breakfast", "Texture Study: Granola & Yoghurt", 620, true}, {31, "Breakfast", "Golden Gallery Pancakes", 730, true}, {32, "Breakfast", "Artisan Breakfast Sandwich", 620, true}, {33, "Breakfast", "Still Life Fruit Salad", 500, true}, {34, "Breakfast", "Warming Tradition Porridge", 450, true},
		{35, "Sandwiches & Wraps", "Classical Ham & Cheese Composition", 620, true}, {36, "Sandwiches & Wraps", "Oceanic Tuna Melt", 680, true}, {37, "Sandwiches & Wraps", "Italian Impression Chicken Pesto", 730, true}, {38, "Sandwiches & Wraps", "Garden Exhibition Veggie Wrap", 620, true}, {39, "Sandwiches & Wraps", "Three-Layer BLT Installation", 680, true}, {40, "Sandwiches & Wraps", "Eastern Influence Falafel Wrap", 680, true}, {41, "Sandwiches & Wraps", "The Curator's Club Sandwich", 850, true}, {42, "Sandwiches & Wraps", "Springtime Egg & Cress Study", 560, true},
		{43, "Salads", "Imperial Caesar Salad", 950, true}, {44, "Salads", "Mediterranean Mosaic Salad", 850, true}, {45, "Salads", "Ancient Grains Quinoa Canvas", 900, true}, {46, "Salads", "Protein Showcase Chicken Salad", 1000, true}, {47, "Salads", "Italian Gallery Pasta Salad", 850, true},
		{48, "Desserts", "Dark Chocolate Sculpture Brownie", 390, true}, {49, "Desserts", "Spiced Masterpiece Carrot Cake", 450, true}, {50, "Desserts", "Cream Gallery Cheesecake", 500, true}, {51, "Desserts", "Heritage Collection Scone", 390, true}, {52, "Desserts", "Baker's Exhibition Muffin", 340, true}, {53, "Desserts", "Victorian Portrait Sponge", 450, true}, {54, "Desserts", "Citrus Showcase Lemon Cake", 450, true}, {55, "Desserts", "French Impressions Macarons", 500, true},
		{56, "Sides", "Crisp Canvas", 170, true}, {57, "Sides", "Mixed Textures Nut Selection", 280, true}, {58, "Sides", "Nature's Palette Fruit Cup", 170, true}, {59, "Sides", "Chocolatier's Choice Bar", 200, true}, {60, "Sides", "Daily Inspiration Soup", 500, true},
	}
}
