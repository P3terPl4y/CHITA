package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type addressCache struct {
	Address string
	Until   time.Time
}
type searchCacheEntry struct {
	Results []GeocodeResult
	Until   time.Time
}
type GeocodeResult struct {
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
type Geocoder struct {
	base   string
	client *http.Client
	mu     sync.Mutex
	cache  map[string]addressCache
	search map[string]searchCacheEntry
	next   time.Time
	busy   bool
}

func NewGeocoder(base string) *Geocoder {
	if base == "" {
		base = "https://photon.komoot.io"
	}
	return &Geocoder{base: strings.TrimRight(base, "/"), client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, cache: make(map[string]addressCache), search: make(map[string]searchCacheEntry)}
}

// Search resolves a place query through Photon. Results are cached and share
// the same bounded provider rate with Reverse to avoid multiplying upstream load.
func (g *Geocoder) Search(ctx context.Context, raw string) ([]GeocodeResult, error) {
	query, err := Text(raw, 3, 120, "La búsqueda")
	if err != nil {
		return nil, err
	}
	key := strings.ToLower(query)
	now := time.Now()
	g.mu.Lock()
	if entry, ok := g.search[key]; ok && entry.Until.After(now) {
		results := append([]GeocodeResult(nil), entry.Results...)
		g.mu.Unlock()
		return results, nil
	}
	if g.busy || g.next.After(now) {
		g.mu.Unlock()
		return nil, Fail(429, "Espera un momento y vuelve a buscar el lugar")
	}
	g.busy = true
	g.next = now.Add(time.Second)
	g.mu.Unlock()
	defer func() { g.mu.Lock(); g.busy = false; g.mu.Unlock() }()

	endpoint, err := url.Parse(g.base + "/api/")
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") {
		return nil, Fail(503, "El proveedor de direcciones no está configurado")
	}
	params := endpoint.Query()
	params.Set("q", query)
	params.Set("limit", "5")
	endpoint.RawQuery = params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "CHITA/1.0 (+https://chita.duohnson.com)")
	request.Header.Set("Accept-Language", "es")
	response, err := g.client.Do(request)
	if err != nil {
		return nil, Fail(503, "No se pudo buscar el lugar; puedes marcarlo en el mapa")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, Fail(503, "El proveedor de búsqueda no está disponible; puedes marcarlo en el mapa")
	}
	var payload struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				Name     string `json:"name"`
				Street   string `json:"street"`
				Number   string `json:"housenumber"`
				District string `json:"district"`
				City     string `json:"city"`
				State    string `json:"state"`
				Country  string `json:"country"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 256*1024)).Decode(&payload); err != nil {
		return nil, Fail(503, "No se pudo interpretar la búsqueda")
	}
	results := make([]GeocodeResult, 0, 5)
	for _, feature := range payload.Features {
		if len(results) == 5 {
			break
		}
		if len(feature.Geometry.Coordinates) < 2 {
			continue
		}
		lng, lat := feature.Geometry.Coordinates[0], feature.Geometry.Coordinates[1]
		if !ValidPoint(lat, lng) {
			continue
		}
		street := strings.TrimSpace(feature.Properties.Street + " " + feature.Properties.Number)
		if street == "" {
			street = feature.Properties.Name
		}
		parts := []string{}
		seen := map[string]bool{}
		for _, part := range []string{street, feature.Properties.District, feature.Properties.City, feature.Properties.State, feature.Properties.Country} {
			part = strings.TrimSpace(part)
			if part != "" && !seen[part] {
				parts = append(parts, part)
				seen[part] = true
			}
		}
		address, textErr := Text(strings.Join(parts, ", "), 3, 255, "Dirección encontrada")
		if textErr == nil {
			results = append(results, GeocodeResult{Address: address, Latitude: lat, Longitude: lng})
		}
	}
	if len(results) == 0 {
		return nil, Fail(404, "No se encontraron lugares; prueba otra búsqueda o coloca el pin manualmente")
	}
	g.mu.Lock()
	if len(g.search) >= 256 {
		g.search = make(map[string]searchCacheEntry)
	}
	g.search[key] = searchCacheEntry{Results: append([]GeocodeResult(nil), results...), Until: time.Now().Add(15 * time.Minute)}
	g.mu.Unlock()
	return results, nil
}

func (g *Geocoder) Reverse(ctx context.Context, lat, lng float64) (string, error) {
	if !ValidPoint(lat, lng) {
		return "", Fail(422, "Coordenadas inválidas")
	}
	key := fmt.Sprintf("%.5f,%.5f", lat, lng)
	now := time.Now()
	g.mu.Lock()
	if entry, ok := g.cache[key]; ok && entry.Until.After(now) {
		g.mu.Unlock()
		return entry.Address, nil
	}
	if g.busy || g.next.After(now) {
		g.mu.Unlock()
		return "", Fail(429, "Espera un momento y vuelve a buscar la dirección")
	}
	g.busy = true
	g.next = now.Add(time.Second)
	g.mu.Unlock()
	defer func() { g.mu.Lock(); g.busy = false; g.mu.Unlock() }()
	endpoint, e := url.Parse(g.base + "/reverse")
	if e != nil || endpoint.Host == "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") {
		return "", Fail(503, "El proveedor de direcciones no está configurado")
	}
	q := endpoint.Query()
	q.Set("lat", fmt.Sprintf("%.7f", lat))
	q.Set("lon", fmt.Sprintf("%.7f", lng))
	q.Set("limit", "1")
	q.Set("radius", "0.1")
	endpoint.RawQuery = q.Encode()
	request, e := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if e != nil {
		return "", e
	}
	request.Header.Set("User-Agent", "CHITA/1.0 (+https://chita.duohnson.com)")
	request.Header.Set("Accept-Language", "es")
	response, e := g.client.Do(request)
	if e != nil {
		return "", Fail(503, "No se pudo obtener la dirección; puedes escribirla manualmente")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return "", Fail(503, "El proveedor de direcciones no está disponible; puedes escribirla manualmente")
	}
	var payload struct {
		Features []struct {
			Properties struct {
				Name     string `json:"name"`
				Street   string `json:"street"`
				Number   string `json:"housenumber"`
				City     string `json:"city"`
				District string `json:"district"`
				State    string `json:"state"`
				Country  string `json:"country"`
			} `json:"properties"`
		} `json:"features"`
	}
	if e = json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&payload); e != nil {
		return "", Fail(503, "No se pudo interpretar la dirección")
	}
	if len(payload.Features) == 0 {
		return "", Fail(404, "No se encontró una dirección en este punto; escríbela manualmente")
	}
	p := payload.Features[0].Properties
	street := strings.TrimSpace(p.Street + " " + p.Number)
	if street == "" {
		street = p.Name
	}
	parts := []string{}
	seen := map[string]bool{}
	for _, piece := range []string{street, p.District, p.City, p.State, p.Country} {
		piece = strings.TrimSpace(piece)
		if piece != "" && !seen[piece] {
			parts = append(parts, piece)
			seen[piece] = true
		}
	}
	address, e := Text(strings.Join(parts, ", "), 3, 255, "Dirección encontrada")
	if e != nil {
		return "", Fail(404, "No se encontró una dirección utilizable; escríbela manualmente")
	}
	g.mu.Lock()
	if len(g.cache) >= 512 {
		g.cache = make(map[string]addressCache)
	}
	g.cache[key] = addressCache{address, time.Now().Add(24 * time.Hour)}
	g.mu.Unlock()
	return address, nil
}
