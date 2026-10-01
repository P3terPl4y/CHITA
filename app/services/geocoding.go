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
type Geocoder struct {
	base   string
	client *http.Client
	mu     sync.Mutex
	cache  map[string]addressCache
	next   time.Time
	busy   bool
}

func NewGeocoder(base string) *Geocoder {
	if base == "" {
		base = "https://photon.komoot.io"
	}
	return &Geocoder{base: strings.TrimRight(base, "/"), client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, cache: make(map[string]addressCache)}
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
