package cache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestDescribeReadsWhatAKeyIs(t *testing.T) {
	c := &Cache{name: CacheDatabase}
	generations := map[string]int64{Languages: 2}

	tests := []struct {
		name string
		key  string
		want Key
	}{
		{
			name: "a value of the current generation",
			key:  "cache:languages:v2:text:ru:id",
			want: Key{Kind: KindEntry, Group: Languages, Entry: "text:ru:id", Editable: true},
		},
		{
			name: "a value of an older generation",
			key:  "cache:languages:v1:all",
			want: Key{Kind: KindStale, Group: Languages, Entry: "all"},
		},
		{
			name: "a group never forgotten is at generation zero",
			key:  "cache:organization:v0:settings",
			want: Key{Kind: KindEntry, Group: Organization, Entry: "settings", Editable: true},
		},
		{
			name: "a generation counter",
			key:  "cache:languages:generation",
			want: Key{Kind: KindGeneration, Group: Languages},
		},
		{
			name: "a session",
			key:  "session:user:0123abcd",
			want: Key{Kind: KindSession, Group: UserSession, Entry: "0123abcd"},
		},
		{
			name: "a rate limit for an IPv6 address",
			key:  "ratelimit:public:2001:db8::1",
			want: Key{Kind: KindRateLimit, Group: "public", Entry: "2001:db8::1"},
		},
		{
			name: "something this server does not write",
			key:  "elsewhere",
			want: Key{Kind: KindOther},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.describe(tt.key, generations)
			tt.want.Name, tt.want.TTL = tt.key, -1
			if got != tt.want {
				t.Errorf("describe(%q) = %+v, want %+v", tt.key, got, tt.want)
			}
		})
	}
}

// Nothing in the session database is edited by hand, even what looks like a
// cached value: an administrator edited there holds roles the database never
// gave them.
func TestDescribeKeepsTheSessionDatabaseReadOnly(t *testing.T) {
	c := &Cache{name: SessionDatabase}
	if key := c.describe("cache:admins:v0:someone", nil); key.Editable {
		t.Error("an administrator in the session database is editable")
	}
}

func TestLiveInspectAndManage(t *testing.T) {
	r := liveRedis(t)
	ctx := context.Background()
	c := r.Cache

	c.Set(ctx, Languages, "all", []string{"en"})
	c.Set(ctx, Organization, "settings", map[string]string{"name": "Acme"})
	r.Sessions.FillSession(ctx, UserSession, "abc", map[string]string{"id": "s"}, time.Now().Add(time.Hour))

	stats, err := c.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Kinds[KindEntry] != 2 || stats.Sessions[UserSession] != 0 {
		t.Errorf("cache stats = %+v, want its two values and none of the sessions", stats)
	}

	sessions, err := r.Sessions.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sessions.Sessions[UserSession] != 1 {
		t.Errorf("session stats = %+v, want the one session", sessions)
	}

	keys, _, err := c.Keys(ctx, KeyQuery{Kind: KindEntry, Group: Languages})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Entry != "all" || keys[0].TTL <= 0 {
		t.Fatalf("keys = %+v, want the languages' one value with its time to live", keys)
	}
	name := keys[0].Name

	if err := c.Write(ctx, name, json.RawMessage(`["en","ru"]`), 0); err != nil {
		t.Fatal(err)
	}
	var all []string
	if !c.Get(ctx, Languages, "all", &all) || len(all) != 2 {
		t.Errorf("Get() after writing by hand = %v, want the written value", all)
	}

	value, err := c.Read(ctx, name)
	if err != nil || string(value.Value) != `["en","ru"]` || value.TTL <= 0 {
		t.Errorf("Read() = %+v, %v; want the value with its time to live kept", value, err)
	}

	if err := r.Sessions.Write(ctx, "session:user:abc", json.RawMessage(`{}`), 0); !errors.Is(err, ErrNotEditable) {
		t.Errorf("writing a session = %v, want ErrNotEditable", err)
	}
	if _, err := c.Delete(ctx, "cache:languages:generation"); !errors.Is(err, ErrGenerationKey) {
		t.Errorf("removing a generation = %v, want ErrGenerationKey", err)
	}

	if removed, err := c.Delete(ctx, name); err != nil || !removed {
		t.Errorf("Delete() = %v, %v; want it removed", removed, err)
	}
	if _, err := c.Read(ctx, name); !errors.Is(err, ErrNoKey) {
		t.Errorf("Read() of a removed key = %v, want ErrNoKey", err)
	}

	if err := c.Clear(ctx, Organization); err != nil {
		t.Fatal(err)
	}
	var settings map[string]string
	if c.Get(ctx, Organization, "settings", &settings) {
		t.Error("a cleared group still answered")
	}

	// Flushing the cache signs nobody out.
	if err := c.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	var session map[string]string
	if !r.Sessions.Session(ctx, UserSession, "abc", &session) {
		t.Error("flushing the cache database removed a session")
	}
}
