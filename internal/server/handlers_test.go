package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/eternalApril/moonlight/internal/config"
	"github.com/eternalApril/moonlight/internal/logger"
	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/storage"
)

// setupEngine creates a fresh engine with a clean store for each test
func setupEngine() *Engine {
	s, _ := storage.NewShardedMapStorage(1) //nolint:errcheck
	eng, _ := NewEngine(s, &config.Config{
		GC: config.GCConfig{Enabled: false},
		Persistence: config.PersistenceConfig{
			AOF: config.AOFConfig{
				Enabled: false,
			},
			RDB: config.RDBConfig{
				Enabled: false,
			},
		},
	}, logger.New("debug", "console"))
	return eng
}

// helper to construct a RESP command request
func makeCommand(_ string, args ...string) []resp.Value {
	vals := make([]resp.Value, len(args))
	for i, arg := range args {
		vals[i] = resp.MakeBulkString(arg)
	}
	return vals
}

func TestPing(t *testing.T) {
	e := setupEngine()

	tests := []struct {
		name     string
		args     []string
		wantType byte
		wantStr  string
		isError  bool
	}{
		{"Simple PING", []string{}, resp.TypeSimpleString, "PONG", false},
		{"PING with message", []string{"Hello"}, resp.TypeBulkString, "Hello", false},
		{"PING too many args", []string{"a", "b"}, resp.TypeError, string(resp.MakeErrorWrongNumberOfArguments("PING").String), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := e.Execute("PING", makeCommand("PING", tt.args...))
			if res.Type != tt.wantType {
				t.Errorf("got type %v, want %v", res.Type, tt.wantType)
			}

			got := string(res.String)
			if got != tt.wantStr {
				t.Errorf("got %q, want %q", got, tt.wantStr)
			}
		})
	}
}

func TestBasicSetGetDel(t *testing.T) {
	e := setupEngine()

	// GET missing key
	res := e.Execute("GET", makeCommand("GET", "mykey"))
	if res.IsNull != true {
		t.Errorf("expected null for missing key, got %v", res.Type)
	}

	// SET key
	res = e.Execute("SET", makeCommand("SET", "mykey", "myvalue"))
	if string(res.String) != "OK" {
		t.Errorf("expected OK, got %v", res.String)
	}

	// GET key
	res = e.Execute("GET", makeCommand("GET", "mykey"))
	if string(res.String) != "myvalue" {
		t.Errorf("expected myvalue, got %s", res.String)
	}

	// DEL key
	res = e.Execute("DEL", makeCommand("DEL", "mykey"))
	if res.Integer != 1 {
		t.Errorf("expected 1 deleted, got %d", res.Integer)
	}

	// GET key again
	res = e.Execute("GET", makeCommand("GET", "mykey"))
	if res.IsNull != true {
		t.Errorf("expected null after delete, got %v", res.Type)
	}
}

func TestSetNX_XX(t *testing.T) {
	e := setupEngine()

	// SET NX on new key -> OK
	res := e.Execute("SET", makeCommand("SET", "k1", "v1", "NX"))
	if string(res.String) != "OK" {
		t.Errorf("SET NX new key failed")
	}

	// SET NX on existing key -> Nil
	res = e.Execute("SET", makeCommand("SET", "k1", "v2", "NX"))
	if res.IsNull != true {
		t.Errorf("SET NX existing key should return nil, got %v", res.Type)
	}
	// Verify value didn't change
	val := e.Execute("GET", makeCommand("GET", "k1"))
	if string(val.String) != "v1" {
		t.Errorf("SET NX changed value despite failure")
	}

	// SET XX on missing key -> Nil
	res = e.Execute("SET", makeCommand("SET", "k2", "v2", "XX"))
	if res.IsNull != true {
		t.Errorf("SET XX missing key should return nil, got %v", res.Type)
	}

	// SET XX on existing key -> OK
	res = e.Execute("SET", makeCommand("SET", "k1", "v_updated", "XX"))
	if string(res.String) != "OK" {
		t.Errorf("SET XX existing key failed")
	}
	val = e.Execute("GET", makeCommand("GET", "k1"))
	if string(val.String) != "v_updated" {
		t.Errorf("SET XX failed to update value")
	}
}

func TestSetTTL(t *testing.T) {
	e := setupEngine()

	// SET EX (Seconds)
	e.Execute("SET", makeCommand("SET", "k_ex", "val", "EX", "1"))

	// Check immediately
	ttl := e.Execute("TTL", makeCommand("TTL", "k_ex"))
	if ttl.Integer != 1 {
		t.Errorf("expected TTL 1, got %d", ttl.Integer)
	}

	// Wait for expiration (1.1s)
	time.Sleep(1100 * time.Millisecond)
	res := e.Execute("GET", makeCommand("GET", "k_ex"))
	if res.IsNull != true {
		t.Errorf("key should have expired")
	}

	// SET PX (Milliseconds)
	e.Execute("SET", makeCommand("SET", "k_px", "val", "PX", "100"))

	pttl := e.Execute("PTTL", makeCommand("PTTL", "k_px"))
	if pttl.Integer <= 0 || pttl.Integer > 100 {
		t.Errorf("expected PTTL ~100ms, got %d", pttl.Integer)
	}

	time.Sleep(150 * time.Millisecond)
	res = e.Execute("GET", makeCommand("GET", "k_px"))
	if res.IsNull != true {
		t.Errorf("key should have expired (PX)")
	}
}

func TestSetKeepTTL(t *testing.T) {
	e := setupEngine()

	// Set key with TTL of 100 seconds
	e.Execute("SET", makeCommand("SET", "k_keep", "v1", "EX", "100"))

	// Update value but Keep TTL
	e.Execute("SET", makeCommand("SET", "k_keep", "v2", "KEEPTTL"))

	val := e.Execute("GET", makeCommand("GET", "k_keep"))
	if string(val.String) != "v2" {
		t.Errorf("KEEPTTL value not updated")
	}

	// Verify TTL is still approx 100
	ttl := e.Execute("TTL", makeCommand("TTL", "k_keep"))
	if ttl.Integer < 95 || ttl.Integer > 100 {
		t.Errorf("KEEPTTL removed the expiration, got %d", ttl.Integer)
	}

	// Verify KEEPTTL on new key behaves like persistent key (no TTL)
	e.Execute("SET", makeCommand("SET", "k_new_keep", "v1", "KEEPTTL"))
	ttl = e.Execute("TTL", makeCommand("TTL", "k_new_keep"))
	if ttl.Integer != -1 {
		t.Errorf("KEEPTTL on new key should have -1 TTL, got %d", ttl.Integer)
	}
}

func TestSetTimestamps(t *testing.T) {
	e := setupEngine()

	// EXAT: expire 2 seconds in future
	future := time.Now().Add(2 * time.Second).Unix()
	futureStr := fmt.Sprintf("%d", future)

	e.Execute("SET", makeCommand("SET", "k_exat", "v", "EXAT", futureStr))

	ttl := e.Execute("TTL", makeCommand("TTL", "k_exat"))
	// Should be 1 or 2 depending on rounding
	if ttl.Integer < 1 || ttl.Integer > 2 {
		t.Errorf("EXAT failed, expected ~2s TTL, got %d", ttl.Integer)
	}
}

func TestTTL_PTTL_Codes(t *testing.T) {
	e := setupEngine()

	// Missing Key -> -2
	res := e.Execute("TTL", makeCommand("TTL", "missing"))
	if res.Integer != -2 {
		t.Errorf("expected -2 for missing key, got %d", res.Integer)
	}

	// Persistent Key -> -1
	e.Execute("SET", makeCommand("SET", "persistent", "val"))
	res = e.Execute("TTL", makeCommand("TTL", "persistent"))
	if res.Integer != -1 {
		t.Errorf("expected -1 for persistent key, got %d", res.Integer)
	}
	res = e.Execute("PTTL", makeCommand("PTTL", "persistent"))
	if res.Integer != -1 {
		t.Errorf("expected -1 for persistent key (PTTL), got %d", res.Integer)
	}
}

func TestSetSyntaxErrors(t *testing.T) {
	e := setupEngine()

	tests := []struct {
		name     string
		args     []string
		expected string // partial error string match
	}{
		{
			"NX and XX together",
			[]string{"k", "v", "NX", "XX"},
			"XX cannot use with NX",
		},
		{
			"XX and NX together",
			[]string{"k", "v", "XX", "NX"},
			"NX cannot use with XX",
		},
		{
			"EX without value",
			[]string{"k", "v", "EX"},
			"syntax error",
		},
		{
			"EX with non-integer",
			[]string{"k", "v", "EX", "abc"},
			"value TTL is not integer",
		},
		{
			"Double TTL (EX then PX)",
			[]string{"k", "v", "EX", "10", "PX", "100"},
			"TTL already specified",
		},
		{
			"KEEPTTL with EX",
			[]string{"k", "v", "KEEPTTL", "EX", "10"},
			"TTL already specified",
		},
		{
			"Unknown Argument",
			[]string{"k", "v", "FOOBAR"},
			"syntax error with command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := e.Execute("SET", makeCommand("SET", tt.args...))
			if res.Type != resp.TypeError {
				t.Errorf("expected error, got %v", res.Type)
			}
			if !strings.Contains(string(res.String), tt.expected) {
				t.Errorf("expected error containing %q, got %q", tt.expected, res.String)
			}
		})
	}
}
