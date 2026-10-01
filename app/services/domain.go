package services

import (
	"fmt"
	"math"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type Problem struct {
	Status  int
	Message string
}

func (p *Problem) Error() string            { return p.Message }
func Fail(status int, message string) error { return &Problem{status, message} }
func ValidPoint(lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && !math.IsInf(lat, 0) && !math.IsInf(lng, 0) && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}
func Text(raw string, min, max int, label string) (string, error) {
	s := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(s)
	if n < min || n > max {
		return "", Fail(422, fmt.Sprintf("%s debe tener entre %d y %d caracteres", label, min, max))
	}
	return s, nil
}
func Email(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || len(s) > 254 {
		return "", Fail(422, "Correo inválido")
	}
	return s, nil
}
func Password(s string) error {
	if utf8.RuneCountInString(s) < 8 || len(s) > 72 {
		return Fail(422, "Contraseña de al menos 8 caracteres y máximo 72 bytes")
	}
	return nil
}
func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, Fail(422, "Los horarios deben incluir fecha, hora y zona horaria")
	}
	return t.UTC(), nil
}
func Currency(s string) bool { return s == "USD" || s == "CUP" || s == "EUR" }
func TrackingActive(status string) bool {
	return status == "accepted" || status == "picked_up" || status == "arrived"
}
func NextState(status, action, role string) (string, error) {
	type transition struct{ from, to, role string }
	allowed := map[string]transition{"accept": {"published", "accepted", "courier"}, "pickup": {"accepted", "picked_up", "courier"}, "arrive": {"picked_up", "arrived", "courier"}, "report": {"arrived", "delivery_reported", "courier"}, "confirm": {"delivery_reported", "completed", "company"}, "reject": {"delivery_reported", "arrived", "company"}}
	if action == "cancel" {
		if role == "company" && (status == "published" || status == "accepted") {
			return "cancelled", nil
		}
		return "", Fail(409, "Sólo puedes cancelar antes de la recogida")
	}
	x, ok := allowed[action]
	if !ok {
		return "", Fail(422, "Acción desconocida")
	}
	if x.role != role {
		return "", Fail(403, "Esta acción corresponde a otro rol")
	}
	if status != x.from {
		return "", Fail(409, "El trabajo cambió de estado; actualiza la página")
	}
	return x.to, nil
}
