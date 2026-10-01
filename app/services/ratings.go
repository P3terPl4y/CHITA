package services

import (
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
)

type RatingInput struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}
type RatingSummary struct {
	CompletedJobs int64                 `json:"completed_jobs"`
	CanRate       bool                  `json:"can_rate"`
	Average       float64               `json:"average"`
	Count         int64                 `json:"count"`
	Rating        *models.CourierRating `json:"rating"`
}

func completedJobs(tx orm.Query, companyID, courierID uint) (int64, error) {
	return tx.WithTrashed().Model(&models.Publication{}).Where("company_id=? AND assigned_courier_id=? AND status='completed' AND confirmed_at IS NOT NULL", companyID, courierID).Count()
}
func RatingInfo(u *models.User, courierID uint) (*RatingSummary, error) {
	if u.Role != "company" && !(u.Role == "courier" && u.ID == courierID) {
		return nil, Fail(403, "No puedes consultar estas calificaciones")
	}
	var target models.User
	if e := facades.Orm().Query().WithTrashed().Where("id=? AND role='courier'", courierID).First(&target); e != nil {
		return nil, e
	}
	if target.ID == 0 {
		return nil, Fail(404, "Repartidor no encontrado")
	}
	out := &RatingSummary{}
	if u.Role == "company" {
		c, e := Company(facades.Orm().Query(), u.ID)
		if e != nil {
			return nil, e
		}
		out.CompletedJobs, e = completedJobs(facades.Orm().Query(), c.ID, courierID)
		if e != nil {
			return nil, e
		}
		out.CanRate = out.CompletedJobs >= 3
		var rating models.CourierRating
		if e = facades.Orm().Query().Where("company_id=? AND courier_user_id=?", c.ID, courierID).First(&rating); e != nil {
			return nil, e
		}
		if rating.ID > 0 {
			out.Rating = &rating
		}
	}
	var aggregate struct {
		Average float64
		Count   int64
	}
	if e := facades.Orm().Query().Raw("SELECT COALESCE(AVG(rating),0) AS average,COUNT(*) AS count FROM courier_ratings WHERE courier_user_id=?", courierID).Scan(&aggregate); e != nil {
		return nil, e
	}
	out.Average = aggregate.Average
	out.Count = aggregate.Count
	return out, nil
}
func RateCourier(u *models.User, courierID uint, r RatingInput) error {
	if u.Role != "company" {
		return Fail(403, "Sólo una empresa puede calificar")
	}
	if r.Rating < 1 || r.Rating > 5 {
		return Fail(422, "La calificación debe estar entre 1 y 5")
	}
	comment, e := Text(r.Comment, 0, 1000, "Comentario")
	if e != nil {
		return e
	}
	return facades.Orm().Transaction(func(tx orm.Query) error {
		if e := activeUser(tx, u.ID, "company"); e != nil {
			return e
		}
		c, e := Company(tx, u.ID)
		if e != nil {
			return e
		}
		var courier models.User
		if e := tx.WithTrashed().Where("id=? AND role='courier'", courierID).LockForUpdate().First(&courier); e != nil {
			return e
		}
		if courier.ID == 0 {
			return Fail(404, "Repartidor no encontrado")
		}
		n, e := completedJobs(tx, c.ID, courierID)
		if e != nil {
			return e
		}
		if n < 3 {
			return Fail(403, "Necesitas tres entregas completadas y confirmadas por esta empresa para calificar")
		}
		var rating models.CourierRating
		if e := tx.Where("company_id=? AND courier_user_id=?", c.ID, courierID).First(&rating); e != nil {
			return e
		}
		if rating.ID == 0 {
			rating = models.CourierRating{CompanyID: c.ID, CourierUserID: courierID, AuthorUserID: u.ID, Rating: r.Rating, Comment: comment}
			e = tx.Create(&rating)
		} else {
			_, e = tx.Model(&models.CourierRating{}).Where("id=?", rating.ID).Update(map[string]any{"rating": r.Rating, "comment": comment, "author_user_id": u.ID})
		}
		if e != nil {
			return e
		}
		return tx.Create(&models.Notification{UserID: courierID, Message: c.TradeName + " actualizó tu calificación: " + string(rune('0'+r.Rating)) + " de 5"})
	})
}
