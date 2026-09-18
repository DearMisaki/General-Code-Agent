package memory

import "testing"

func TestNewMemoryStoreFromConfigUsesLocalWhenDisabled(t *testing.T) {
	store, err := NewMemoryStoreFromConfig(MemoryConfig{
		LongTerm: LongTermMemoryConfig{Enabled: false},
	})
	if err != nil {
		t.Fatalf("NewMemoryStoreFromConfig() = %v", err)
	}
	if _, ok := store.(*LocalMemoryStore); !ok {
		t.Fatalf("store type = %T, want *LocalMemoryStore", store)
	}
}

func TestNewMemoryStoreFromConfigRequiresAPIKeyForMem0(t *testing.T) {
	_, err := NewMemoryStoreFromConfig(MemoryConfig{
		LongTerm: LongTermMemoryConfig{Enabled: true, Backend: "mem0-platform"},
	})
	if err == nil {
		t.Fatal("mem0 backend without API key should fail")
	}
}
