package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"sync"
	"time"
)

// launchTokenLife is how long a launch token can be used for: long
// enough for a browser that has to start first, short enough that one
// that never arrives is soon worth nothing.
const launchTokenLife = 2 * time.Minute

// launchTokens are the one-time tokens a browser is opened with, the way
// Jupyter Server opens its page. The program that opens a browser, and
// often the browser itself, shows its command line to every account on
// the machine, so the page's key is never put there. A launch token is
// exchanged on its first use for a cookie that holds this run's session,
// not the key, and is then, or after launchTokenLife, no key at all. Whoever
// uses it first, which may be another account that read it off that
// command line, gets no more than that session, which ends when the server
// does. They are kept in memory only.
type launchTokens struct {
	mu     sync.Mutex
	tokens map[string]time.Time // token -> when it stops working
	now    func() time.Time
}

func newLaunchTokens() *launchTokens {
	return &launchTokens{tokens: map[string]time.Time{}, now: time.Now}
}

// randomSecret is 32 random bytes as 64 hex characters, for a launch
// token or a session.
func randomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand does not fail on any system Go supports; a secret
		// that could be guessed must never be handed out regardless.
		panic("web: no random bytes for a secret: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// issue makes a new launch token.
func (l *launchTokens) issue() string {
	tok := randomSecret()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune()
	l.tokens[tok] = l.now().Add(launchTokenLife)
	return tok
}

// redeem reports whether tok is a launch token that is still good, and
// uses it up if it is, so of any number of requests carrying the same
// token exactly one is let through. Every token held is compared in
// constant time, and an empty one never matches.
func (l *launchTokens) redeem(tok string) bool {
	if l == nil || tok == "" {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune()
	found := ""
	for t := range l.tokens {
		if subtle.ConstantTimeCompare([]byte(t), []byte(tok)) == 1 {
			found = t
		}
	}
	if found == "" {
		return false
	}
	delete(l.tokens, found)
	return true
}

// prune drops the tokens whose time is up. The caller holds mu.
func (l *launchTokens) prune() {
	now := l.now()
	for t, until := range l.tokens {
		if !now.Before(until) {
			delete(l.tokens, t)
		}
	}
}
