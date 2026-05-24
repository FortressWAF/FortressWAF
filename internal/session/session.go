package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

type Session struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id,omitempty"`
	RealIP       string                 `json:"real_ip"`
	UserAgent    string                 `json:"user_agent"`
	Country      string                 `json:"country"`
	ASN          uint                   `json:"asn"`
	CreatedAt    time.Time              `json:"created_at"`
	LastSeen     time.Time              `json:"last_seen"`
	RequestCount int                    `json:"request_count"`
	RiskScore    float64                `json:"risk_score"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	Blocked      bool                   `json:"blocked"`
	Decisions    []sessionDecision      `json:"decisions,omitempty"`
	Fingerprint  string                 `json:"fingerprint,omitempty"`
}

type sessionDecision struct {
	RuleID    string    `json:"rule_id"`
	Action    string    `json:"action"`
	Score     float64   `json:"score"`
	Timestamp time.Time `json:"timestamp"`
}

type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	byUser   map[string]string
	byIP     map[string][]string
	ttl      time.Duration
}

func NewStore(ttl time.Duration) *Store {
	s := &Store{
		sessions: make(map[string]*Session),
		byUser:   make(map[string]string),
		byIP:     make(map[string][]string),
		ttl:      ttl,
	}
	go s.cleanupLoop()
	return s
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) Create(ip, userAgent string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		ID:        generateID(),
		RealIP:    ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	s.sessions[session.ID] = session
	s.byIP[ip] = append(s.byIP[ip], session.ID)

	return session
}

func (s *Store) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

func (s *Store) GetOrCreate(id, ip, userAgent string) *Session {
	if id != "" {
		if session := s.Get(id); session != nil {
			s.updateLastSeen(session)
			return session
		}
	}
	return s.Create(ip, userAgent)
}

func (s *Store) updateLastSeen(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session.LastSeen = time.Now()
	session.RequestCount++
}

func (s *Store) Update(id string, fn func(*Session)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[id]
	if !exists {
		return
	}

	fn(session)
	session.LastSeen = time.Now()
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.sessions[id]; exists {
		if userSID, ok := s.byUser[session.UserID]; ok && userSID == id {
			delete(s.byUser, session.UserID)
		}
		if ips, ok := s.byIP[session.RealIP]; ok {
			filtered := make([]string, 0, len(ips))
			for _, sid := range ips {
				if sid != id {
					filtered = append(filtered, sid)
				}
			}
			if len(filtered) == 0 {
				delete(s.byIP, session.RealIP)
			} else {
				s.byIP[session.RealIP] = filtered
			}
		}
		delete(s.sessions, id)
	}
}

func (s *Store) GetByUser(userID string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if id, ok := s.byUser[userID]; ok {
		return s.sessions[id]
	}
	return nil
}

func (s *Store) GetByIP(ip string) []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.byIP[ip]
	result := make([]*Session, 0, len(ids))
	for _, id := range ids {
		if session, ok := s.sessions[id]; ok {
			result = append(result, session)
		}
	}
	return result
}

func (s *Store) AddRiskScore(id string, score float64, ruleID, action string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[id]
	if !exists {
		return
	}

	session.RiskScore += score
	session.Decisions = append(session.Decisions, sessionDecision{
		RuleID:    ruleID,
		Action:    action,
		Score:     score,
		Timestamp: time.Now(),
	})

	if session.RiskScore > 100 {
		session.RiskScore = 100
	}
}

func (s *Store) DetectSessionFixation(newSessionID, oldSessionID string) bool {
	if oldSessionID != "" && newSessionID != "" {
		oldSession := s.Get(oldSessionID)
		if oldSession != nil && time.Since(oldSession.CreatedAt) < time.Minute {
			return true
		}
	}
	return false
}

func (s *Store) Serialize() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.sessions)
}

func (s *Store) Deserialize(data []byte) error {
	var sessions map[string]*Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = sessions

	for id, session := range sessions {
		s.byIP[session.RealIP] = append(s.byIP[session.RealIP], id)
		if session.UserID != "" {
			s.byUser[session.UserID] = id
		}
	}

	return nil
}

func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, session := range s.sessions {
			if now.Sub(session.LastSeen) > s.ttl {
				if userSID, ok := s.byUser[session.UserID]; ok && userSID == id {
					delete(s.byUser, session.UserID)
				}
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

func (s *Store) ActiveUsers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byUser)
}

var _ = slog.Debug
