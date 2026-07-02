package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

// --- Data structures (from v1 Implementation Contract) ---

type Pomodoro struct {
	StartedAt time.Time `json:"-"`
	EndsAt    time.Time `json:"-"`
}

type Planned struct {
	StartMin    int `json:"-"`
	EndMin      int `json:"-"`
	DurationMin int `json:"duration_min"`
}

type Actual struct {
	Start        *string  `json:"start,omitempty"`
	End          *string  `json:"end,omitempty"`
	OverrunMin   int      `json:"overrun_min"`
	FocusQuality *float64 `json:"focus_quality,omitempty"`
}

type Block struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	StartMin  int       `json:"-"`
	EndMin    int       `json:"-"`
	Planned   Planned   `json:"planned"`
	Actual    Actual    `json:"actual"`
	Pomodoro  *Pomodoro `json:"-"`
	StartStr  string    `json:"start"`
	EndStr    string    `json:"end"`
}

type Day struct {
	Date   string  `json:"date"`
	Blocks []Block `json:"blocks"`
}

// --- Schedule template ---

var dhaka = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		return time.FixedZone("BDT", 6*3600)
	}
	return loc
}()

const pomodoroDuration = 25 * time.Minute

func weekdaySchedule() []Block {
	return []Block{
		{ID: "workout", Label: "Workout", StartMin: 6*60 + 15, EndMin: 7 * 60},
		{ID: "breakfast", Label: "Shower / Breakfast", StartMin: 7 * 60, EndMin: 8 * 60},
		{ID: "go-block", Label: "Go / AI Learning Block", StartMin: 8 * 60, EndMin: 9*60 + 30},
		{ID: "work-am", Label: "Work (Morning)", StartMin: 9*60 + 30, EndMin: 13 * 60},
		{ID: "lunch", Label: "Lunch", StartMin: 13 * 60, EndMin: 14 * 60},
		{ID: "work-pm", Label: "Work (Afternoon)", StartMin: 14 * 60, EndMin: 18 * 60},
		{ID: "wind-down", Label: "Evening Wind-down", StartMin: 18 * 60, EndMin: 19 * 60},
	}
}

func restDaySchedule() []Block {
	return []Block{
		{ID: "rest", Label: "Rest Day", StartMin: 0, EndMin: 23*60 + 59},
	}
}

func buildDay(date string) *Day {
	t, err := time.ParseInLocation("2006-01-02", date, dhaka)
	if err != nil {
		return &Day{Date: date, Blocks: nil}
	}
	var blocks []Block
	switch t.Weekday() {
	case time.Friday, time.Saturday:
		blocks = restDaySchedule()
	default:
		blocks = weekdaySchedule()
	}
	for i := range blocks {
		b := &blocks[i]
		b.Planned = Planned{StartMin: b.StartMin, EndMin: b.EndMin, DurationMin: b.EndMin - b.StartMin}
		b.StartStr = minToHHMM(b.StartMin)
		b.EndStr = minToHHMM(b.EndMin)
	}
	return &Day{Date: date, Blocks: blocks}
}

func minToHHMM(m int) string {
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

// --- Current block logic ---

func nowMinutes(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

func currentBlock(day *Day, t time.Time) *Block {
	if day == nil {
		return nil
	}
	m := nowMinutes(t)
	for i := range day.Blocks {
		b := &day.Blocks[i]
		if m >= b.StartMin && m < b.EndMin {
			return b
		}
	}
	return nil
}

func secondsRemainingInBlock(b *Block, t time.Time) int {
	if b == nil {
		return 0
	}
	endMin := b.EndMin
	nowSec := t.Hour()*3600 + t.Minute()*60 + t.Second()
	endSec := endMin * 60
	rem := endSec - nowSec
	if rem < 0 {
		return 0
	}
	return rem
}

func nextBlockStart(day *Day, t time.Time) (int, bool) {
	if day == nil {
		return 0, false
	}
	m := nowMinutes(t)
	for i := range day.Blocks {
		b := &day.Blocks[i]
		if b.StartMin > m {
			return b.StartMin, true
		}
	}
	return 0, false
}

func pomodoroSecondsRemaining(b *Block, t time.Time) *int {
	if b == nil || b.Pomodoro == nil {
		return nil
	}
	rem := int(b.Pomodoro.EndsAt.Sub(t).Seconds())
	if rem < 0 {
		rem = 0
	}
	return &rem
}

// --- In-memory store with lazy-eval daily reset ---

type Store struct {
	mu   sync.RWMutex
	days map[string]*Day
}

func newStore() *Store {
	return &Store{days: make(map[string]*Day)}
}

func (s *Store) getOrBuild(date string) *Day {
	s.mu.Lock()
	defer s.mu.Unlock()
	if day, ok := s.days[date]; ok {
		return day
	}
	day := buildDay(date)
	s.days[date] = day
	return day
}

// snapshotDay returns a deep-enough copy of the Day safe to marshal without
// holding the lock. Pointer fields (*string, *float64) point to values that
// are never mutated in place (always reassigned), so a shallow copy of each
// Block under RLock gives a consistent snapshot.
func (s *Store) snapshotDay(date string) *Day {
	s.mu.RLock()
	defer s.mu.RUnlock()
	day := s.days[date]
	if day == nil {
		return nil
	}
	cp := *day
	cp.Blocks = make([]Block, len(day.Blocks))
	for i := range day.Blocks {
		cp.Blocks[i] = day.Blocks[i]
	}
	return &cp
}

// pomoRemain reads a block's Pomodoro remaining seconds under RLock so it
// doesn't race with handleFocusStart's write to block.Pomodoro.
func (s *Store) pomoRemain(block *Block, now time.Time) *int {
	if block == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if block.Pomodoro == nil {
		return nil
	}
	rem := int(block.Pomodoro.EndsAt.Sub(now).Seconds())
	if rem < 0 {
		rem = 0
	}
	return &rem
}

func (s *Store) getBlock(date, blockID string) (*Block, *Day) {
	day := s.getOrBuild(date)
	for i := range day.Blocks {
		if day.Blocks[i].ID == blockID {
			return &day.Blocks[i], day
		}
	}
	return nil, day
}

// --- WebSocket Hub (E1) ---

type Hub struct {
	mu         sync.Mutex
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 16),
		register:   make(chan *websocket.Conn, 4),
		unregister: make(chan *websocket.Conn, 4),
	}
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.Lock()
			for conn := range h.clients {
				conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
				if err := conn.WriteMessage(1, msg); err != nil {
					delete(h.clients, conn)
					conn.Close()
				}
			}
			h.mu.Unlock()
		}
	}
}

// --- Tick message ---

type TickMsg struct {
	Type                     string `json:"type"`
	Date                     string `json:"date"`
	BlockID                  *string `json:"block_id"`
	SecondsRemainingInBlock  *int    `json:"seconds_remaining_in_block"`
	PomodoroSecondsRemaining *int    `json:"pomodoro_seconds_remaining"`
	NextBlockStart           *string `json:"next_block_start"`
}

// --- Block-transition side effects (E4: tick goroutine lazily builds new day) ---

type Server struct {
	store     *Store
	hub       *Hub
	prevBlock string
	prevDate  string
}

func newServer() *Server {
	return &Server{
		store: newStore(),
		hub:   newHub(),
	}
}

func (srv *Server) tickLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().In(dhaka)
		date := now.Format("2006-01-02")

		// Reset prevBlock when the date changes so we never stamp a bogus
		// Actual.End on the new day's same-named block (latent cross-day bug).
		if srv.prevDate != date {
			srv.prevBlock = ""
			srv.prevDate = date
		}

		day := srv.store.getOrBuild(date)
		block := currentBlock(day, now)

		var blockID *string
		var secRem *int
		if block != nil {
			id := block.ID
			blockID = &id
			s := secondsRemainingInBlock(block, now)
			secRem = &s

			if block.ID != srv.prevBlock {
				srv.store.mu.Lock()
				if srv.prevBlock != "" {
					if prev, _ := day.findBlock(srv.prevBlock); prev != nil {
						ts := now.Format("15:04:05")
						prev.Actual.End = &ts
					}
				}
				ts := now.Format("15:04:05")
				block.Actual.Start = &ts
				srv.prevBlock = block.ID
				srv.store.mu.Unlock()
			}
		} else {
			if srv.prevBlock != "" {
				srv.store.mu.Lock()
				if prev, _ := day.findBlock(srv.prevBlock); prev != nil {
					ts := now.Format("15:04:05")
					prev.Actual.End = &ts
				}
				srv.prevBlock = ""
				srv.store.mu.Unlock()
			}
		}

		// Read Pomodoro state under RLock (was racing handleFocusStart's write).
		pomoRem := srv.store.pomoRemain(block, now)

		// Compute next block start for free-time hint (server Dhaka time, not client).
		var nextStart *string
		if block == nil {
			if ns, ok := nextBlockStart(day, now); ok {
				s := minToHHMM(ns)
				nextStart = &s
			}
		}

		msg := TickMsg{
			Type:                     "tick",
			Date:                     date,
			BlockID:                  blockID,
			SecondsRemainingInBlock:  secRem,
			PomodoroSecondsRemaining: pomoRem,
			NextBlockStart:           nextStart,
		}
		data, _ := json.Marshal(msg)
		select {
		case srv.hub.broadcast <- data:
		default:
		}
	}
}

func (d *Day) findBlock(id string) (*Block, *Day) {
	for i := range d.Blocks {
		if d.Blocks[i].ID == id {
			return &d.Blocks[i], d
		}
	}
	return nil, d
}

// --- Block JSON serialization (HH:MM from minutes) ---

func (b Block) MarshalJSON() ([]byte, error) {
	type alias struct {
		ID       string  `json:"id"`
		Label    string  `json:"label"`
		Start    string  `json:"start"`
		End      string  `json:"end"`
		Planned  Planned `json:"planned"`
		Actual   Actual  `json:"actual"`
		HasPomo  bool    `json:"has_pomodoro"`
	}
	a := alias{
		ID:      b.ID,
		Label:   b.Label,
		Start:   b.StartStr,
		End:     b.EndStr,
		Planned: b.Planned,
		Actual:  b.Actual,
		HasPomo: b.Pomodoro != nil,
	}
	return json.Marshal(a)
}

// --- HTTP handlers ---

func (srv *Server) handleToday(c *fiber.Ctx) error {
	now := time.Now().In(dhaka)
	date := now.Format("2006-01-02")
	// Try a read-locked snapshot first; build if missing, then snapshot again.
	day := srv.store.snapshotDay(date)
	if day == nil {
		srv.store.getOrBuild(date)
		day = srv.store.snapshotDay(date)
	}
	return c.JSON(day)
}

func (srv *Server) handleFocusStart(c *fiber.Ctx) error {
	var body struct {
		BlockID string `json:"block_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.BlockID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "block_id required"})
	}
	now := time.Now().In(dhaka)
	date := now.Format("2006-01-02")
	block, _ := srv.store.getBlock(date, body.BlockID)
	if block == nil {
		return c.Status(404).JSON(fiber.Map{"error": "block not found"})
	}
	srv.store.mu.Lock()
	endsAt := now.Add(pomodoroDuration)
	block.Pomodoro = &Pomodoro{
		StartedAt: now,
		EndsAt:    endsAt,
	}
	srv.store.mu.Unlock()
	return c.JSON(fiber.Map{
		"started_at": now.Format(time.RFC3339),
		"ends_at":    endsAt.Format(time.RFC3339),
	})
}

func (srv *Server) handleFocusStop(c *fiber.Ctx) error {
	now := time.Now().In(dhaka)
	date := now.Format("2006-01-02")
	day := srv.store.getOrBuild(date)
	srv.store.mu.Lock()
	defer srv.store.mu.Unlock()
	// Search all blocks for an active pomodoro, not just the current one —
	// a pomodoro started on a block that has since ended still needs clearing.
	for i := range day.Blocks {
		if day.Blocks[i].Pomodoro != nil {
			day.Blocks[i].Pomodoro = nil
			return c.JSON(fiber.Map{"ok": true})
		}
	}
	return c.Status(409).JSON(fiber.Map{"error": "no active pomodoro"})
}

func (srv *Server) handleActual(c *fiber.Ctx) error {
	var body struct {
		BlockID    string `json:"block_id"`
		OverrunMin int    `json:"overrun_min"`
	}
	if err := c.BodyParser(&body); err != nil || body.BlockID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "block_id required"})
	}
	if body.OverrunMin < 0 {
		return c.Status(400).JSON(fiber.Map{"error": "overrun_min must be >= 0"})
	}
	now := time.Now().In(dhaka)
	date := now.Format("2006-01-02")
	block, _ := srv.store.getBlock(date, body.BlockID)
	if block == nil {
		return c.Status(404).JSON(fiber.Map{"error": "block not found"})
	}
	srv.store.mu.Lock()
	defer srv.store.mu.Unlock()
	if block.Actual.End == nil {
		return c.Status(409).JSON(fiber.Map{"error": "block not finished"})
	}
	block.Actual.OverrunMin = body.OverrunMin
	return c.JSON(fiber.Map{"ok": true})
}

func (srv *Server) handleWS(c *websocket.Conn) {
	srv.hub.register <- c
	defer func() {
		srv.hub.unregister <- c
	}()
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

func main() {
	srv := newServer()
	go srv.hub.run()
	go srv.tickLoop()

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,OPTIONS",
	}))

	app.Get("/api/today", srv.handleToday)
	app.Post("/api/focus/start", srv.handleFocusStart)
	app.Post("/api/focus/stop", srv.handleFocusStop)
	app.Post("/api/actual", srv.handleActual)

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", websocket.New(srv.handleWS))

	log.Fatal(app.Listen(":3000"))
}
