package compliance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type AuditEntry struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	ActorID     string    `json:"actor_id"`
	ActorType   string    `json:"actor_type"`
	ActorIP     string    `json:"actor_ip"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	ResourceID  string    `json:"resource_id"`
	Result      string    `json:"result"`
	Metadata    string    `json:"metadata,omitempty"`
	Hash        string    `json:"hash"`
	PrevHash    string    `json:"prev_hash"`
}

type AuditLog struct {
	mu           sync.RWMutex
	entries      []AuditEntry
	lastHash     string
	immutable    bool
}

func NewAuditLog() *AuditLog {
	return &AuditLog{
		entries:   make([]AuditEntry, 0),
		immutable: true,
	}
}

func (al *AuditLog) Append(entry AuditEntry) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.immutable {
		return fmt.Errorf("audit log is immutable")
	}

	entry.Timestamp = time.Now()
	entry.ID = fmt.Sprintf("audit-%d", len(al.entries)+1)
	entry.PrevHash = al.lastHash

	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
		entry.ID, entry.Timestamp.Format(time.RFC3339Nano),
		entry.ActorID, entry.ActorType, entry.ActorIP,
		entry.Action, entry.Resource, entry.ResourceID, entry.Result)

	hash := sha256.New()
	hash.Write([]byte(al.lastHash + data))
	entry.Hash = hex.EncodeToString(hash.Sum(nil))

	al.entries = append(al.entries, entry)
	al.lastHash = entry.Hash

	return nil
}

func (al *AuditLog) Query(filter AuditFilter) ([]AuditEntry, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var results []AuditEntry
	for _, entry := range al.entries {
		if filter.ActorID != "" && entry.ActorID != filter.ActorID {
			continue
		}
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}
		if filter.Resource != "" && entry.Resource != filter.Resource {
			continue
		}
		if filter.From.After(entry.Timestamp) || filter.To.Before(entry.Timestamp) {
			continue
		}
		results = append(results, entry)
	}

	return results, nil
}

type AuditFilter struct {
	ActorID   string
	Action    string
	Resource  string
	From      time.Time
	To        time.Time
}

func (al *AuditLog) VerifyIntegrity() (bool, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var prevHash string
	for i, entry := range al.entries {
		if i == 0 && entry.PrevHash != "" {
			return false, fmt.Errorf("chain broken at entry %d: first entry has prev_hash", i)
		}
		if entry.PrevHash != prevHash {
			return false, fmt.Errorf("chain broken at entry %d: expected %s, got %s", i, prevHash, entry.PrevHash)
		}
		prevHash = entry.Hash
	}

	return true, nil
}