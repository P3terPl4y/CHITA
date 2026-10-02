package services

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	_ "image/png"
	"strings"
	"unicode/utf8"

	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
)

// NormalizeAvatar decodes bounded raster images and re-encodes them without metadata.
// No user-supplied paths, remote URLs, SVG or original file bytes are stored.
func NormalizeAvatar(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	prefix, encoded, ok := strings.Cut(value, ",")
	if !ok || (prefix != "data:image/jpeg;base64" && prefix != "data:image/png;base64") || len(encoded) > 28000 {
		return "", Fail(422, "Usa una foto JPG o PNG pequeña")
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(raw) > 20000 {
		return "", Fail(422, "La foto no es válida o es demasiado grande")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "jpeg" && format != "png") || config.Width < 1 || config.Height < 1 || config.Width > 256 || config.Height > 256 {
		return "", Fail(422, "La foto debe medir como máximo 256 × 256")
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", Fail(422, "No se pudo leer la foto")
	}
	var output bytes.Buffer
	if err = jpeg.Encode(&output, img, &jpeg.Options{Quality: 80}); err != nil {
		return "", err
	}
	if output.Len() > 20000 {
		return "", Fail(422, "Reduce el tamaño de la foto")
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(output.Bytes()), nil
}

func SaveAvatar(u *models.User, value, grant string) (string, error) {
	avatar, err := NormalizeAvatar(value)
	if err != nil {
		return "", err
	}
	err = facades.Orm().Transaction(func(tx orm.Query) error {
		if e := activeUser(tx, u.ID, u.Role); e != nil {
			return e
		}
		if e := LockGrant(tx, u.ID, grant); e != nil {
			return e
		}
		_, e := tx.Exec("UPDATE users SET avatar_url=?,admin_version=admin_version+1 WHERE id=? AND deleted_at IS NULL", avatar, u.ID)
		return e
	})
	return avatar, err
}

type DirectoryCourier struct {
	ID            uint    `json:"id"`
	Name          string  `json:"name"`
	AvatarURL     string  `json:"avatar_url"`
	VehicleType   string  `json:"vehicle_type"`
	AverageRating float64 `json:"average_rating"`
	RatingCount   int64   `json:"rating_count"`
	Membership    string  `json:"membership"`
}
type CourierDirectory struct {
	Items []DirectoryCourier `json:"items"`
	Total int64              `json:"total"`
}

func Directory(u *models.User, search string, page int) (CourierDirectory, error) {
	out := CourierDirectory{Items: []DirectoryCourier{}}
	if u.Role != "company" {
		return out, Fail(403, "Sólo las empresas pueden consultar el directorio")
	}
	if page < 1 || page > 100000 || utf8.RuneCountInString(search) > 100 {
		return out, Fail(422, "Búsqueda o página no válida")
	}
	company, err := Company(facades.Orm().Query(), u.ID)
	if err != nil {
		return out, err
	}
	pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.TrimSpace(search)) + "%"
	condition := " FROM users u JOIN courier_profiles p ON p.user_id=u.id WHERE u.role='courier' AND u.status AND u.deleted_at IS NULL AND u.display_name ILIKE ?"
	var count struct{ Total int64 }
	if err = facades.Orm().Query().Raw("SELECT COUNT(*) AS total"+condition, pattern).Scan(&count); err != nil {
		return out, err
	}
	out.Total = count.Total
	err = facades.Orm().Query().Raw(`SELECT u.id,u.display_name AS name,u.avatar_url,p.vehicle_type,COALESCE((SELECT AVG(r.rating) FROM courier_ratings r WHERE r.courier_user_id=u.id),0) AS average_rating,(SELECT COUNT(*) FROM courier_ratings r WHERE r.courier_user_id=u.id) AS rating_count,COALESCE((SELECT m.status FROM company_members m WHERE m.company_id=? AND m.user_id=u.id),'none') AS membership`+condition+" ORDER BY lower(u.display_name),u.id LIMIT 30 OFFSET ?", company.ID, pattern, (page-1)*30).Scan(&out.Items)
	return out, err
}
func InviteCourier(u *models.User, courierID uint) (*models.CompanyMember, error) {
	if u.Role != "company" {
		return nil, Fail(403, "Sólo la empresa puede invitar")
	}
	var courier models.User
	if err := facades.Orm().Query().Where("id=?", courierID).Where("role=?", "courier").Where("status=?", true).First(&courier); err != nil {
		return nil, err
	}
	if courier.ID == 0 {
		return nil, Fail(404, "Repartidor no encontrado")
	}
	return Invite(u, courier.Email)
}
