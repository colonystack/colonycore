package core

import (
	"testing"
	"time"

	"colonycore/pkg/domain"
)

func TestClockFuncNowNilFallsBackToUTCTime(t *testing.T) {
	got := ClockFunc(nil).Now()
	if got.IsZero() {
		t.Fatal("expected non-zero time from nil ClockFunc")
	}
	if got.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %s", got.Location())
	}
}

func TestClockFuncNowDelegatesToFunction(t *testing.T) {
	expected := time.Date(2024, 7, 4, 12, 34, 56, 0, time.FixedZone("offset", -5*3600))
	fn := ClockFunc(func() time.Time { return expected })
	got := fn.Now()
	if !got.Equal(expected.UTC()) {
		t.Fatalf("expected %s, got %s", expected.UTC(), got)
	}
}

func TestExtractRulesEngine(t *testing.T) {
	engine := domain.NewRulesEngine()
	store := NewMemoryStore(engine)
	if got := extractRulesEngine(store); got != engine {
		t.Fatalf("expected engine pointer, got %v", got)
	}
	if extractRulesEngine(&fakePersistentStore{}) != nil {
		t.Fatal("expected nil for stores without RulesEngine provider")
	}
}

func TestSelectNowFuncPrefersStoreProvider(t *testing.T) {
	expected := time.Date(2025, 1, 2, 3, 4, 5, 0, time.FixedZone("cet", 3600))
	store := &providerStore{
		fakePersistentStore: &fakePersistentStore{},
		engine:              domain.NewRulesEngine(),
		now:                 func() time.Time { return expected },
	}
	nowFn := selectNowFunc(store, nil)
	if got := nowFn(); !got.Equal(expected.UTC()) {
		t.Fatalf("expected store now func to be used, got %s", got)
	}
}

func TestSelectNowFuncFallsBackToClock(t *testing.T) {
	expected := time.Date(2030, 5, 6, 7, 8, 9, 0, time.UTC)
	clock := ClockFunc(func() time.Time { return expected })
	store := &providerStore{
		fakePersistentStore: &fakePersistentStore{},
		engine:              domain.NewRulesEngine(),
	}
	nowFn := selectNowFunc(store, clock)
	if got := nowFn(); !got.Equal(expected) {
		t.Fatalf("expected clock fallback, got %s", got)
	}
}

func TestSelectNowFuncDefaultsToSystemUTC(t *testing.T) {
	store := &fakePersistentStore{}
	nowFn := selectNowFunc(store, nil)
	got := nowFn()
	if got.Location() != time.UTC {
		t.Fatalf("expected UTC time, got %s", got.Location())
	}
	if time.Since(got) > time.Second || time.Since(got) < -time.Second {
		t.Fatalf("expected near-current time, got %s", got)
	}
}

type providerStore struct {
	*fakePersistentStore
	engine *domain.RulesEngine
	now    func() time.Time
}

func (p *providerStore) RulesEngine() *domain.RulesEngine { return p.engine }

func (p *providerStore) NowFunc() func() time.Time { return p.now }
