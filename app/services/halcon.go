package services

import (
	"encoding/json"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type RemoteCookie struct {
	Name  string
	Value string
}
type RemoteSession struct{ Cookies []RemoteCookie }
type Remote struct {
	Base   string
	Client *http.Client
}
type Point struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}
type Location struct {
	Latitude  *float64   `json:"latitude"`
	Longitude *float64   `json:"longitude"`
	Active    bool       `json:"active"`
	LastSeen  *time.Time `json:"last_seen"`
}

func NewRemote(raw string) (*Remote, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("HALCON_URL inválida")
	}
	ip := net.ParseIP(u.Hostname())
	loop := u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
	if u.Scheme != "https" && !(u.Scheme == "http" && loop) {
		return nil, fmt.Errorf("HALCON_URL requiere HTTPS o HTTP de loopback")
	}
	return &Remote{Base: strings.TrimSuffix(raw, "/"), Client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (r *Remote) call(s *RemoteSession, method, path string, data url.Values, out any) error {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}
	req, e := http.NewRequest(method, r.Base+path, body)
	if e != nil {
		return e
	}
	req.Header.Set("Origin", r.Base)
	if data != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, c := range s.Cookies {
		req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value})
	}
	res, e := r.Client.Do(req)
	if e != nil {
		return Fail(502, "HALCON no está disponible")
	}
	defer res.Body.Close()
	for _, c := range res.Cookies() {
		found := false
		for i := range s.Cookies {
			if s.Cookies[i].Name == c.Name {
				s.Cookies[i].Value = c.Value
				found = true
			}
		}
		if !found {
			s.Cookies = append(s.Cookies, RemoteCookie{c.Name, c.Value})
		}
	}
	if res.StatusCode == 401 || res.StatusCode == 403 {
		return Fail(409, "La sesión de HALCON venció o las credenciales no son válidas")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Fail(502, "HALCON rechazó la solicitud")
	}
	if out != nil && json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(out) != nil {
		return Fail(502, "Respuesta inesperada de HALCON")
	}
	return nil
}
func (r *Remote) Login(email, password string) (RemoteSession, uint, uint, error) {
	var s RemoteSession
	var auth struct {
		CSRF string `json:"csrf_token"`
		User struct {
			ID   uint   `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	if e := r.call(&s, "GET", "/api/session", nil, &auth); e != nil {
		return s, 0, 0, e
	}
	if e := r.call(&s, "POST", "/api/auth/login", url.Values{"email": {email}, "password": {password}, "_csrf": {auth.CSRF}}, &auth); e != nil {
		return s, 0, 0, e
	}
	if auth.User.ID == 0 || auth.User.Role != "user" {
		r.Logout(s)
		return RemoteSession{}, 0, 0, Fail(422, "Vincula una cuenta personal de HALCON, sin permisos de moderación")
	}
	var track struct {
		PersonalID uint `json:"personal_id"`
	}
	if e := r.call(&s, "GET", "/api/tracking", nil, &track); e != nil {
		r.Logout(s)
		return RemoteSession{}, 0, 0, e
	}
	if track.PersonalID == 0 {
		r.Logout(s)
		return RemoteSession{}, 0, 0, Fail(409, "Tu cuenta de HALCON no tiene un halcón personal")
	}
	return s, auth.User.ID, track.PersonalID, nil
}
func (r *Remote) Logout(s RemoteSession) {
	var a struct {
		CSRF string `json:"csrf_token"`
	}
	if r.call(&s, "GET", "/api/session", nil, &a) == nil {
		_ = r.call(&s, "POST", "/api/auth/logout", url.Values{"_csrf": {a.CSRF}}, nil)
	}
}
func (r *Remote) Location(s RemoteSession, id uint) (Location, error) {
	var out struct {
		Halcones []struct {
			ID        uint       `json:"id"`
			Latitude  *float64   `json:"LastLat"`
			Longitude *float64   `json:"LastLng"`
			Active    bool       `json:"IsActive"`
			LastSeen  *time.Time `json:"LastSeen"`
		} `json:"halcones"`
	}
	if e := r.call(&s, "GET", "/api/tracking", nil, &out); e != nil {
		return Location{}, e
	}
	for _, p := range out.Halcones {
		if p.ID == id {
			if p.LastSeen == nil {
				return Location{}, nil
			}
			return Location{p.Latitude, p.Longitude, p.Active && p.LastSeen != nil && time.Since(*p.LastSeen) < 60*time.Second, p.LastSeen}, nil
		}
	}
	return Location{}, Fail(409, "El halcón vinculado ya no está disponible")
}
func (r *Remote) Dial(s RemoteSession) (*websocket.Conn, error) {
	u, _ := url.Parse(r.Base)
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/location"
	headers := http.Header{"Origin": {r.Base}}
	req := http.Request{Header: headers}
	for _, c := range s.Cookies {
		req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value})
	}
	d := websocket.Dialer{HandshakeTimeout: 8 * time.Second}
	c, res, e := d.Dial(u.String(), req.Header)
	if res != nil && res.Body != nil {
		res.Body.Close()
	}
	if e != nil {
		return nil, Fail(502, "No se pudo abrir el seguimiento en HALCON")
	}
	return c, nil
}

type stream struct {
	conn *websocket.Conn
	job  uint
	last time.Time
}
type Tracking struct {
	Remote  *Remote
	mu      sync.Mutex
	streams map[uint]*stream
	done    chan struct{}
	once    sync.Once
	locks   [128]sync.Mutex
}

func NewTracking(r *Remote) *Tracking {
	t := &Tracking{Remote: r, streams: map[uint]*stream{}, done: make(chan struct{})}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-t.done:
				return
			case <-ticker.C:
				t.mu.Lock()
				for uid, s := range t.streams {
					if time.Since(s.last) > 45*time.Second {
						s.conn.Close()
						delete(t.streams, uid)
					}
				}
				t.mu.Unlock()
			}
		}
	}()
	return t
}
func (t *Tracking) Close() {
	t.once.Do(func() {
		close(t.done)
		t.mu.Lock()
		defer t.mu.Unlock()
		for id, s := range t.streams {
			s.conn.Close()
			delete(t.streams, id)
		}
	})
}
func (t *Tracking) Stop(uid uint) { t.StopJob(uid, 0) }
func (t *Tracking) StopJob(uid, job uint) {
	lock := &t.locks[uid%128]
	lock.Lock()
	defer lock.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	if s := t.streams[uid]; s != nil && (job == 0 || job == s.job) {
		s.conn.Close()
		delete(t.streams, uid)
	}
}
func account(q orm.Query, uid uint) (*models.HalconAccount, error) {
	var a models.HalconAccount
	if e := q.Where("user_id=?", uid).First(&a); e != nil {
		return nil, e
	}
	if a.ID == 0 || !a.ExpiresAt.After(time.Now()) {
		return nil, Fail(409, "El repartidor debe vincular una sesión vigente de HALCON")
	}
	return &a, nil
}
func decrypt(a *models.HalconAccount) (RemoteSession, error) {
	raw, e := facades.Crypt().DecryptString(a.SessionCiphertext)
	if e != nil {
		return RemoteSession{}, e
	}
	var s RemoteSession
	e = json.Unmarshal([]byte(raw), &s)
	return s, e
}
func (t *Tracking) Info(u *models.User) (any, error) {
	if u.Role != "courier" {
		return nil, Fail(403, "La vinculación corresponde al repartidor")
	}
	var a models.HalconAccount
	if e := facades.Orm().Query().Where("user_id=?", u.ID).First(&a); e != nil {
		return nil, e
	}
	return map[string]any{"linked": a.ID != 0 && a.ExpiresAt.After(time.Now()), "expires_at": a.ExpiresAt}, nil
}
func (t *Tracking) Link(u *models.User, email, password string) error {
	if u.Role != "courier" {
		return Fail(403, "La vinculación corresponde al repartidor")
	}
	if _, e := Email(email); e != nil {
		return e
	}
	if len(password) == 0 || len(password) > 72 {
		return Fail(422, "Contraseña inválida")
	}
	s, remoteID, personalID, e := t.Remote.Login(email, password)
	if e != nil {
		return e
	}
	raw, _ := json.Marshal(s)
	cipher, e := facades.Crypt().EncryptString(string(raw))
	if e != nil {
		t.Remote.Logout(s)
		return e
	}
	var old models.HalconAccount
	e = facades.Orm().Transaction(func(tx orm.Query) error {
		var user models.User
		if e := tx.Where("id=?", u.ID).LockForUpdate().First(&user); e != nil {
			return e
		}
		if user.ID == 0 || !user.Status {
			return Fail(401, "Tu sesión terminó")
		}
		var conflict models.HalconAccount
		if e := tx.Where("halcon_user_id=?", remoteID).Where("user_id<>?", u.ID).First(&conflict); e != nil {
			return e
		}
		if conflict.ID != 0 {
			return Fail(409, "Esta cuenta de HALCON ya está vinculada a otro repartidor")
		}
		if e := tx.Where("user_id=?", u.ID).First(&old); e != nil {
			return e
		}
		values := map[string]any{"halcon_user_id": remoteID, "personal_id": personalID, "session_ciphertext": cipher, "expires_at": time.Now().UTC().Add(25 * time.Minute)}
		if old.ID != 0 {
			_, e := tx.Model(&models.HalconAccount{}).Where("id=?", old.ID).Update(values)
			return e
		}
		return tx.Create(&models.HalconAccount{UserID: u.ID, HalconUserID: remoteID, PersonalID: personalID, SessionCiphertext: cipher, ExpiresAt: time.Now().UTC().Add(25 * time.Minute)})
	})
	if e != nil {
		t.Remote.Logout(s)
		return e
	}
	t.Stop(u.ID)
	if old.ID != 0 {
		if s, e := decrypt(&old); e == nil {
			t.Remote.Logout(s)
		}
	}
	return nil
}
func (t *Tracking) Unlink(u *models.User) error {
	if u.Role != "courier" {
		return Fail(403, "La vinculación corresponde al repartidor")
	}
	var a models.HalconAccount
	err := facades.Orm().Transaction(func(tx orm.Query) error {
		var user models.User
		if err := tx.Where("id=?", u.ID).LockForUpdate().First(&user); err != nil {
			return err
		}
		if err := tx.Where("user_id=?", u.ID).LockForUpdate().First(&a); err != nil {
			return err
		}
		if a.ID == 0 {
			return nil
		}
		_, err := tx.Delete(&a)
		return err
	})
	if err != nil {
		return err
	}
	t.Stop(u.ID)
	if a.ID != 0 {
		if s, e := decrypt(&a); e == nil {
			t.Remote.Logout(s)
		}
	}
	return nil
}
func trackingJob(q orm.Query, u *models.User, id uint) (*models.Publication, error) {
	var p models.Publication
	if e := q.Where("id=?", id).First(&p); e != nil {
		return nil, e
	}
	if p.ID == 0 || p.AssignedCourierID == nil {
		return nil, Fail(404, "Seguimiento no encontrado")
	}
	if u.Role == "company" {
		c, e := Company(facades.Orm().Query(), u.ID)
		if e != nil || c.ID != p.CompanyID {
			return nil, Fail(404, "Seguimiento no encontrado")
		}
	} else if *p.AssignedCourierID != u.ID {
		return nil, Fail(404, "Seguimiento no encontrado")
	}
	if !TrackingActive(p.Status) {
		return nil, Fail(409, "El seguimiento sólo está disponible durante el trabajo activo")
	}
	return &p, nil
}
func (t *Tracking) Publish(u *models.User, id uint, p Point, grant string) error {
	if u.Role != "courier" {
		return Fail(403, "Sólo el repartidor asignado puede enviar su ubicación")
	}
	if p.Latitude == nil || p.Longitude == nil || !ValidPoint(*p.Latitude, *p.Longitude) {
		return Fail(422, "Coordenadas inválidas")
	}
	return facades.Orm().Transaction(func(tx orm.Query) error {
		if err := LockGrant(tx, u.ID, grant); err != nil {
			return err
		}
		if _, e := trackingJob(tx.LockForUpdate(), u, id); e != nil {
			return e
		}
		a, e := account(tx.LockForUpdate(), u.ID)
		if e != nil {
			return e
		}
		s, e := decrypt(a)
		if e != nil {
			return e
		}
		lock := &t.locks[u.ID%128]
		lock.Lock()
		defer lock.Unlock()
		t.mu.Lock()
		st := t.streams[u.ID]
		if st != nil && st.job != id {
			st.conn.Close()
			delete(t.streams, u.ID)
			st = nil
		}
		t.mu.Unlock()
		if st == nil {
			c, e := t.Remote.Dial(s)
			if e != nil {
				return e
			}
			st = &stream{conn: c, job: id, last: time.Now()}
			t.mu.Lock()
			select {
			case <-t.done:
				t.mu.Unlock()
				c.Close()
				return Fail(503, "Seguimiento detenido")
			default:
			}
			t.streams[u.ID] = st
			t.mu.Unlock()
			go t.read(u.ID, st)
		}
		st.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		payload, _ := json.Marshal(p)
		if e := st.conn.WriteMessage(websocket.TextMessage, payload); e != nil {
			st.conn.Close()
			t.mu.Lock()
			delete(t.streams, u.ID)
			t.mu.Unlock()
			return Fail(502, "HALCON cerró el seguimiento; vuelve a intentarlo")
		}
		t.mu.Lock()
		st.last = time.Now()
		t.mu.Unlock()
		return nil
	})
}
func (t *Tracking) Position(u *models.User, id uint) (Location, error) {
	p, e := trackingJob(facades.Orm().Query(), u, id)
	if e != nil {
		return Location{}, e
	}
	a, e := account(facades.Orm().Query(), *p.AssignedCourierID)
	if e != nil {
		return Location{}, e
	}
	s, e := decrypt(a)
	if e != nil {
		return Location{}, e
	}
	loc, e := t.Remote.Location(s, a.PersonalID)
	if e != nil {
		return Location{}, e
	}
	if _, e := trackingJob(facades.Orm().Query(), u, id); e != nil {
		return Location{}, e
	}
	current, err := account(facades.Orm().Query(), *p.AssignedCourierID)
	if err != nil {
		return Location{}, err
	}
	if current.ID != a.ID || current.SessionCiphertext != a.SessionCiphertext {
		return Location{}, Fail(409, "La vinculación de HALCON cambió")
	}
	return loc, nil
}

func (t *Tracking) read(uid uint, st *stream) {
	defer st.conn.Close()
	for {
		if _, _, err := st.conn.ReadMessage(); err != nil {
			break
		}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.streams[uid] == st {
		delete(t.streams, uid)
	}
}
