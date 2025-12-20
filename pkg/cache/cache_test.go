package cache

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New()
	defer c.Close()

	if c == nil {
		t.Fatal("Expected non-nil cache")
	}

	if c.ttl != DefaultTTL {
		t.Errorf("Expected TTL %v, got %v", DefaultTTL, c.ttl)
	}
}

func TestNewWithTTL(t *testing.T) {
	customTTL := 10 * time.Minute
	c := New(WithTTL(customTTL))
	defer c.Close()

	if c.ttl != customTTL {
		t.Errorf("Expected TTL %v, got %v", customTTL, c.ttl)
	}
}

func TestSetGet(t *testing.T) {
	c := New()
	defer c.Close()

	c.Set("key1", "value1")
	c.Set("key2", 42)
	c.Set("key3", []string{"a", "b", "c"})

	// Test string value
	if v, ok := c.Get("key1"); !ok || v != "value1" {
		t.Errorf("Expected 'value1', got %v (ok=%v)", v, ok)
	}

	// Test int value
	if v, ok := c.Get("key2"); !ok || v != 42 {
		t.Errorf("Expected 42, got %v (ok=%v)", v, ok)
	}

	// Test slice value
	if v, ok := c.Get("key3"); !ok {
		t.Error("Expected to find key3")
	} else {
		slice, ok := v.([]string)
		if !ok || len(slice) != 3 {
			t.Errorf("Expected []string{a,b,c}, got %v", v)
		}
	}

	// Test non-existent key
	if _, ok := c.Get("nonexistent"); ok {
		t.Error("Expected not to find nonexistent key")
	}
}

func TestGetString(t *testing.T) {
	c := New()
	defer c.Close()

	c.Set("str", "hello")
	c.Set("int", 42)

	// Test string value
	if v, ok := c.GetString("str"); !ok || v != "hello" {
		t.Errorf("Expected 'hello', got %v (ok=%v)", v, ok)
	}

	// Test non-string value
	if _, ok := c.GetString("int"); ok {
		t.Error("Expected GetString to fail for non-string value")
	}
}

func TestDelete(t *testing.T) {
	c := New()
	defer c.Close()

	c.Set("key", "value")
	c.Delete("key")

	if _, ok := c.Get("key"); ok {
		t.Error("Expected key to be deleted")
	}
}

func TestClear(t *testing.T) {
	c := New()
	defer c.Close()

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Clear()

	if c.Len() != 0 {
		t.Errorf("Expected 0 items after clear, got %d", c.Len())
	}
}

func TestLen(t *testing.T) {
	c := New()
	defer c.Close()

	if c.Len() != 0 {
		t.Errorf("Expected 0, got %d", c.Len())
	}

	c.Set("key1", "value1")
	c.Set("key2", "value2")

	if c.Len() != 2 {
		t.Errorf("Expected 2, got %d", c.Len())
	}
}

func TestExpiration(t *testing.T) {
	c := New(WithTTL(50 * time.Millisecond))
	defer c.Close()

	c.Set("key", "value")

	// Should exist immediately
	if _, ok := c.Get("key"); !ok {
		t.Error("Expected key to exist")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	if _, ok := c.Get("key"); ok {
		t.Error("Expected key to be expired")
	}
}

func TestSetWithTTL(t *testing.T) {
	c := New(WithTTL(1 * time.Hour)) // Long default TTL
	defer c.Close()

	// Set with short TTL
	c.SetWithTTL("short", "value", 50*time.Millisecond)
	c.Set("long", "value")

	time.Sleep(100 * time.Millisecond)

	// Short TTL should be expired
	if _, ok := c.Get("short"); ok {
		t.Error("Expected short TTL key to be expired")
	}

	// Long TTL should still exist
	if _, ok := c.Get("long"); !ok {
		t.Error("Expected long TTL key to still exist")
	}
}

func TestGlobalCache(t *testing.T) {
	SetGlobal("gkey", "gvalue")

	if v, ok := GetGlobal("gkey"); !ok || v != "gvalue" {
		t.Errorf("Expected 'gvalue', got %v (ok=%v)", v, ok)
	}

	if v, ok := GetGlobalString("gkey"); !ok || v != "gvalue" {
		t.Errorf("Expected 'gvalue', got %v (ok=%v)", v, ok)
	}

	DeleteGlobal("gkey")
	if _, ok := GetGlobal("gkey"); ok {
		t.Error("Expected global key to be deleted")
	}

	SetGlobal("key1", "value1")
	SetGlobal("key2", "value2")
	ClearGlobal()

	if _, ok := GetGlobal("key1"); ok {
		t.Error("Expected global cache to be cleared")
	}
}

func TestConcurrency(t *testing.T) {
	c := New()
	defer c.Close()

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			c.Set("key", i)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			c.Get("key")
		}
		done <- true
	}()

	<-done
	<-done
}
