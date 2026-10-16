package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

const maxGeoResponseBytes int64 = 1 << 20

// GeoProvider performs one geolocation lookup.
type GeoProvider interface {
	Lookup(ctx context.Context, ipAddress string) (models.GeoLocation, error)
}

// GeoLocator tries providers in order. It returns the first usable result.
type GeoLocator struct {
	providers []GeoProvider
}

func NewGeoLocator(providers ...GeoProvider) *GeoLocator {
	return &GeoLocator{providers: providers}
}

func (g *GeoLocator) Lookup(
	ctx context.Context,
	ipAddress string,
) (models.GeoLocation, error) {
	ip := net.ParseIP(strings.TrimSpace(ipAddress))
	if ip == nil {
		return models.GeoLocation{}, errors.New("invalid IP address")
	}

	// Never send loopback, private or other non-public addresses
	// to external geolocation services.
	if isNonPublicIP(ip) {
		return models.GeoLocation{}, nil
	}

	if g == nil || len(g.providers) == 0 {
		return models.GeoLocation{}, errors.New("no geolocation providers configured")
	}

	var failures []error

	for index, provider := range g.providers {
		if provider == nil {
			failures = append(
				failures,
				fmt.Errorf("provider %d is nil", index+1),
			)
			continue
		}

		location, err := provider.Lookup(ctx, ip.String())
		if err != nil {
			failures = append(
				failures,
				fmt.Errorf("provider %d: %w", index+1, err),
			)
			continue
		}

		location.Country = strings.TrimSpace(location.Country)
		location.City = strings.TrimSpace(location.City)

		if location.Country == "" && location.City == "" {
			failures = append(
				failures,
				fmt.Errorf("provider %d returned no location data", index+1),
			)
			continue
		}

		return location, nil
	}

	return models.GeoLocation{}, fmt.Errorf(
		"all geolocation providers failed: %w",
		errors.Join(failures...),
	)
}

func isNonPublicIP(ip net.IP) bool {
	return ip.IsPrivate() ||
		ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

func newGeoHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}

	return &http.Client{Timeout: 2500 * time.Millisecond}
}

func getJSON(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	target any,
) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return fmt.Errorf("create geolocation request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "nez-widget-platform/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request geolocation provider: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"geolocation provider returned HTTP %d",
			resp.StatusCode,
		)
	}

	body, err := io.ReadAll(
		io.LimitReader(resp.Body, maxGeoResponseBytes+1),
	)
	if err != nil {
		return fmt.Errorf("read geolocation response: %w", err)
	}
	if int64(len(body)) > maxGeoResponseBytes {
		return errors.New("geolocation response is too large")
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode geolocation response: %w", err)
	}

	return nil
}

// IPAPIProvider uses the ip-api.com JSON endpoint.
// baseURL is configurable so tests can use a local HTTP test server.
type IPAPIProvider struct {
	client  *http.Client
	baseURL string
}

func NewIPAPIProvider(
	client *http.Client,
	baseURL string,
) *IPAPIProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://ip-api.com/json"
	}

	return &IPAPIProvider{
		client:  newGeoHTTPClient(client),
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (p *IPAPIProvider) Lookup(
	ctx context.Context,
	ipAddress string,
) (models.GeoLocation, error) {
	ip := net.ParseIP(strings.TrimSpace(ipAddress))
	if ip == nil {
		return models.GeoLocation{}, errors.New("invalid IP address")
	}

	endpoint, err := url.JoinPath(p.baseURL, ip.String())
	if err != nil {
		return models.GeoLocation{}, fmt.Errorf("build IP-API URL: %w", err)
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return models.GeoLocation{}, fmt.Errorf("parse IP-API URL: %w", err)
	}

	query := parsedURL.Query()
	query.Set("fields", "status,message,country,city")
	parsedURL.RawQuery = query.Encode()

	var response struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Country string `json:"country"`
		City    string `json:"city"`
	}

	if err := getJSON(ctx, p.client, parsedURL.String(), &response); err != nil {
		return models.GeoLocation{}, err
	}

	if response.Status != "success" {
		return models.GeoLocation{}, fmt.Errorf(
			"IP-API lookup failed: %s",
			response.Message,
		)
	}

	location := models.GeoLocation{
		Country: response.Country,
		City:    response.City,
	}
	if strings.TrimSpace(location.Country) == "" &&
		strings.TrimSpace(location.City) == "" {
		return models.GeoLocation{}, errors.New("IP-API returned empty location")
	}

	return location, nil
}

// IPAPICoProvider uses ipapi.co's HTTPS JSON endpoint.
type IPAPICoProvider struct {
	client  *http.Client
	baseURL string
}

func NewIPAPICoProvider(
	client *http.Client,
	baseURL string,
) *IPAPICoProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://ipapi.co"
	}

	return &IPAPICoProvider{
		client:  newGeoHTTPClient(client),
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (p *IPAPICoProvider) Lookup(
	ctx context.Context,
	ipAddress string,
) (models.GeoLocation, error) {
	ip := net.ParseIP(strings.TrimSpace(ipAddress))
	if ip == nil {
		return models.GeoLocation{}, errors.New("invalid IP address")
	}

	endpoint, err := url.JoinPath(p.baseURL, ip.String(), "json")
	if err != nil {
		return models.GeoLocation{}, fmt.Errorf("build ipapi.co URL: %w", err)
	}

	var response struct {
		CountryName string `json:"country_name"`
		Country     string `json:"country"`
		City        string `json:"city"`
		Error       bool   `json:"error"`
		Reason      string `json:"reason"`
	}

	if err := getJSON(ctx, p.client, endpoint, &response); err != nil {
		return models.GeoLocation{}, err
	}

	if response.Error {
		return models.GeoLocation{}, fmt.Errorf(
			"ipapi.co lookup failed: %s",
			response.Reason,
		)
	}

	country := strings.TrimSpace(response.CountryName)
	if country == "" {
		country = strings.TrimSpace(response.Country)
	}

	location := models.GeoLocation{
		Country: country,
		City:    response.City,
	}
	if location.Country == "" && strings.TrimSpace(location.City) == "" {
		return models.GeoLocation{}, errors.New("ipapi.co returned empty location")
	}

	return location, nil
}
