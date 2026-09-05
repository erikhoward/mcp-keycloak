package keycloak

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/Nerzal/gocloak/v14"
)

func TestListAdminEventsUsesBoundedPages(t *testing.T) {
	var mu sync.Mutex
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/realms/master/protocol/openid-connect/token" {
			writeJSON(t, w, map[string]any{"access_token": "test-token", "expires_in": 300})
			return
		}
		if r.URL.Path != "/admin/realms/acme/admin-events" {
			http.NotFound(w, r)
			return
		}
		first, _ := strconv.Atoi(r.URL.Query().Get("first"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("max"))
		mu.Lock()
		requests = append(requests, fmt.Sprintf("%d:%d", first, pageSize))
		mu.Unlock()

		count := pageSize
		if first == 100 {
			count = 50
		}
		events := make([]gocloak.AdminEventRepresentation, count)
		for i := range events {
			events[i].OperationType = gocloak.StringP("UPDATE")
		}
		writeJSON(t, w, events)
	}))
	defer server.Close()

	admin := NewAdmin(server.URL, "admin", "password", "master")
	events, err := admin.ListAdminEvents(t.Context(), "acme", gocloak.GetAdminEventsParams{Max: gocloak.Int32P(150)})
	if err != nil {
		t.Fatalf("ListAdminEvents: %v", err)
	}
	if len(events) != 150 {
		t.Fatalf("got %d events, want 150", len(events))
	}
	mu.Lock()
	defer mu.Unlock()
	if got := fmt.Sprint(requests); got != "[0:100 100:50]" {
		t.Errorf("page requests = %s, want [0:100 100:50]", got)
	}
}

func TestListEventsHonorsSmallMax(t *testing.T) {
	var requestedMax string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/realms/master/protocol/openid-connect/token" {
			writeJSON(t, w, map[string]any{"access_token": "test-token", "expires_in": 300})
			return
		}
		if r.URL.Path != "/admin/realms/acme/events" {
			http.NotFound(w, r)
			return
		}
		requestedMax = r.URL.Query().Get("max")
		events := []gocloak.EventRepresentation{
			{Type: gocloak.StringP("LOGIN")},
			{Type: gocloak.StringP("LOGIN")},
			{Type: gocloak.StringP("LOGIN")},
		}
		writeJSON(t, w, events)
	}))
	defer server.Close()

	admin := NewAdmin(server.URL, "admin", "password", "master")
	events, err := admin.ListEvents(t.Context(), "acme", gocloak.GetEventsParams{Max: gocloak.Int32P(3)})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 3 || requestedMax != "3" {
		t.Errorf("got %d events with max %q, want 3 events and max=3", len(events), requestedMax)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode JSON response: %v", err)
	}
}
