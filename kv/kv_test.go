package kv

import "testing"

func TestStoreApplySetGet(t *testing.T) {
	s := NewStore()
	cmd := Command{Op: OpSet, Key: "a", Value: "1"}
	if err := s.Apply(cmd); err != nil {
		t.Fatalf("apply set failed: %v", err)
	}
	val, ok := s.Get("a")
	if !ok || val != "1" {
		t.Fatalf("expected value 1, got %q", val)
	}
}

func TestStoreApplyInsertConflict(t *testing.T) {
	s := NewStore()
	if err := s.Apply(Command{Op: OpSet, Key: "a", Value: "1"}); err != nil {
		t.Fatalf("apply set failed: %v", err)
	}
	if err := s.Apply(Command{Op: OpInsert, Key: "a", Value: "2"}); err == nil {
		t.Fatalf("expected insert conflict error")
	}
}

func TestStoreApplyDelete(t *testing.T) {
	s := NewStore()
	_ = s.Apply(Command{Op: OpSet, Key: "a", Value: "1"})
	if err := s.Apply(Command{Op: OpDelete, Key: "a"}); err != nil {
		t.Fatalf("apply delete failed: %v", err)
	}
	if _, ok := s.Get("a"); ok {
		t.Fatalf("expected key deleted")
	}
}

func TestCommandEncodeDecode(t *testing.T) {
	cmd := Command{Op: OpSet, Key: "a", Value: "1"}
	data, err := EncodeCommand(cmd)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	decoded, err := DecodeCommand(data)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded != cmd {
		t.Fatalf("expected %#v, got %#v", cmd, decoded)
	}
}

func TestStoreSnapshotRestore(t *testing.T) {
	s := NewStore()
	if err := s.Apply(Command{Op: OpSet, Key: "a", Value: "1"}); err != nil {
		t.Fatalf("apply set failed: %v", err)
	}
	if err := s.Apply(Command{Op: OpSet, Key: "b", Value: "2"}); err != nil {
		t.Fatalf("apply set failed: %v", err)
	}
	snap, err := s.Snapshot()
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	next := NewStore()
	if err := next.RestoreSnapshot(snap); err != nil {
		t.Fatalf("restore snapshot failed: %v", err)
	}
	if val, ok := next.Get("a"); !ok || val != "1" {
		t.Fatalf("expected key a to restore, got %q", val)
	}
	if val, ok := next.Get("b"); !ok || val != "2" {
		t.Fatalf("expected key b to restore, got %q", val)
	}
}

func TestStoreRestoreEmptySnapshotClearsState(t *testing.T) {
	s := NewStore()
	_ = s.Apply(Command{Op: OpSet, Key: "a", Value: "1"})
	if err := s.RestoreSnapshot(nil); err != nil {
		t.Fatalf("restore empty snapshot failed: %v", err)
	}
	if _, ok := s.Get("a"); ok {
		t.Fatalf("expected store to be cleared")
	}
}
