package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

type geoProviderFunc func(
	context.Context,
	string,
) (models.GeoLocation, error)

func (f geoProviderFunc) Lookup(
	ctx context.Context,
	ip string,
) (models.GeoLocation, error) {
	return f(ctx, ip)
}

func TestGeoLocatorFallsBackWhenPrimaryFails(t *testing.T) {
	var primaryCalls, fallbackCalls int

	primary := geoProviderFunc(func(
		_ context.Context,
		ip string,
	) (models.GeoLocation, error) {
		primaryCalls++
		if ip != "8.8.8.8" {
			t.Fatalf("unexpected IP: %q", ip)
		}
		return models.GeoLocation{}, errors.New("primary unavailable")
	})

	fallback := geoProviderFunc(func(
		_ context.Context,
		ip string,
	) (models.GeoLocation, error) {
		fallbackCalls++
		return models.GeoLocation{
			Country: "Nigeria",
			City:    "Lagos",
		}, nil
	})

	locator := NewGeoLocator(primary, fallback)
	got, err := locator.Lookup(context.Background(), "8.8.8.8")

	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if got.Country != "Nigeria" || got.City != "Lagos" {
		t.Fatalf("unexpected location: %+v", got)
	}
	if primaryCalls != 1 || fallbackCalls != 1 {
		t.Fatalf(
			"expected one call per provider; primary=%d fallback=%d",
			primaryCalls,
			fallbackCalls,
		)
	}
}

func TestGeoLocatorReturnsErrorWhenAllProvidersFail(t *testing.T) {
	provider := geoProviderFunc(func(
		_ context.Context,
		_ string,
	) (models.GeoLocation, error) {
		return models.GeoLocation{}, errors.New("provider unavailable")
	})

	locator := NewGeoLocator(provider, provider)

	_, err := locator.Lookup(context.Background(), "8.8.8.8")
	if err == nil {
		t.Fatal("expected an error when all providers fail")
	}
}

func TestGeoLocatorSkipsPrivateIP(t *testing.T) {
	var providerCalls int

	provider := geoProviderFunc(func(
		_ context.Context,
		_ string,
	) (models.GeoLocation, error) {
		providerCalls++
		return models.GeoLocation{}, errors.New("should not be called")
	})

	locator := NewGeoLocator(provider, provider)

	got, err := locator.Lookup(context.Background(), "192.168.1.10")
	if err != nil {
		t.Fatalf("private IP should not cause a lookup failure: %v", err)
	}
	if got.Country != "" || got.City != "" {
		t.Fatalf("expected empty location, got %+v", got)
	}
	if providerCalls != 0 {
		t.Fatal("private IP must not be sent to providers")
	}
}

func TestIPAPIProviderLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/json/8.8.8.8" {
				t.Errorf("unexpected request path: %s", r.URL.Path)
			}
			if r.URL.Query().Get("fields") != "status,message,country,city" {
				t.Errorf("unexpected fields query: %s", r.URL.RawQuery)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"country": "Nigeria",
				"city":    "Lagos",
			})
		},
	))
	defer server.Close()

	provider := NewIPAPIProvider(server.Client(), server.URL+"/json")

	got, err := provider.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if got.Country != "Nigeria" || got.City != "Lagos" {
		t.Fatalf("unexpected location: %+v", got)
	}
}

func TestIPAPICoProviderLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/8.8.8.8/json" {
				t.Errorf("unexpected request path: %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"country_name": "Nigeria",
				"city":         "Lagos",
			})
		},
	))
	defer server.Close()

	provider := NewIPAPICoProvider(server.Client(), server.URL)

	got, err := provider.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if got.Country != "Nigeria" || got.City != "Lagos" {
		t.Fatalf("unexpected location: %+v", got)
	}
}
