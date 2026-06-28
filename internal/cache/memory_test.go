package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemory_SetGetExpiry(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(time.Minute)

	if _, ok, _ := c.Get(ctx, "missing"); ok {
		t.Fatal("expected miss for unset key")
	}

	if err := c.Set(ctx, "k", []byte("v"), 50*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	b, ok, _ := c.Get(ctx, "k")
	if !ok || string(b) != "v" {
		t.Fatalf("expected hit 'v', got ok=%v val=%q", ok, b)
	}

	time.Sleep(70 * time.Millisecond)
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Fatal("expected entry to expire")
	}
}

func TestMemory_Delete(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(time.Minute)
	_ = c.Set(ctx, "k", []byte("v"), time.Minute)
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Fatal("expected miss after delete")
	}
}

func TestGetSetJSON(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(time.Minute)
	type item struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	want := []item{{Name: "a", N: 1}, {Name: "b", N: 2}}
	if err := SetJSON(ctx, c, "list", want, time.Minute); err != nil {
		t.Fatalf("setjson: %v", err)
	}
	got, ok := GetJSON[[]item](ctx, c, "list")
	if !ok || len(got) != 2 || got[1].Name != "b" {
		t.Fatalf("getjson roundtrip failed: ok=%v got=%+v", ok, got)
	}
}
