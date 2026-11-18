package api

import (
	"github.com/KyAnhVo/brainlog-website/server/internal/apitime"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
)

// Wire shapes from requirements/api-shape.md and requirements/Login, account.md.
// Field names and optionality must match the TypeScript types in the clients' lib/*.ts.

type apiError struct {
	ErrorCode int    `json:"errorCode"`
	ErrorMsg  string `json:"errorMsg"`
}

type statusCustomerDTO struct {
	ID         string `json:"id"`
	StatusType string `json:"statusType"`
	Status     string `json:"status"`
	// Present (possibly empty) for good statuses, absent for bad ones.
	Warnings      *[]string `json:"warnings,omitempty"`
	OccurringTime string    `json:"occurringTime"`
	Location      string    `json:"location"`
}

type statusDTO struct {
	statusCustomerDTO
	CreatorID    string `json:"creatorId"`
	CreationTime string `json:"creationTime"`
}

type dimensionsDTO struct {
	LengthCm float64 `json:"lengthCm"`
	WidthCm  float64 `json:"widthCm"`
	HeightCm float64 `json:"heightCm"`
}

type physicalDetailsDTO struct {
	PcsCount   *int          `json:"pcsCount,omitempty"`
	WeightKg   float64       `json:"weightKg"`
	Dimensions dimensionsDTO `json:"dimensions"`
}

type trackingBaseDTO struct {
	TrackingNumber  string              `json:"trackingNumber"`
	Cont            string              `json:"cont,omitempty"`
	Seal            string              `json:"seal,omitempty"`
	POL             string              `json:"pol"`
	POD             string              `json:"pod"`
	PhysicalDetails *physicalDetailsDTO `json:"physicalDetails,omitempty"`
}

// Omit<Tracking, 'lastStatus'>, used by the admin single-tracking view.
type trackingDetailsDTO struct {
	trackingBaseDTO
	CreatorID    string `json:"creatorId"`
	CreationTime string `json:"creationTime"`
	CustomerID   string `json:"customerId,omitempty"`
}

type trackingDTO struct {
	trackingDetailsDTO
	LastStatus statusDTO `json:"lastStatus"`
}

// GET /api/tracking/:trackingNumber
type trackingResponse struct {
	Details  trackingBaseDTO     `json:"details"`
	Statuses []statusCustomerDTO `json:"statuses"`
}

// GET /api/admin/tracking/:trackingNumber
type adminTrackingResponse struct {
	Details  trackingDetailsDTO `json:"details"`
	Statuses []statusDTO        `json:"statuses"`
}

// GET /api/admin/tracking
type trackingListResponse struct {
	Trackings []trackingDTO `json:"trackings"`
	Page      int           `json:"page"`
	PageCount int           `json:"pageCount"`
	PageSize  int           `json:"pageSize"`
}

type loginSuccessResponse struct {
	Name           string `json:"name"`
	AccountCreated string `json:"account_created"`
	Role           string `json:"role"`
}

type accountPublicDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type empty struct{}

func toStatusCustomer(s model.Status) statusCustomerDTO {
	d := statusCustomerDTO{
		ID:            s.ID,
		StatusType:    string(s.Type),
		Status:        s.Status,
		OccurringTime: apitime.Format(s.OccurringTime),
		Location:      s.Location,
	}
	if s.Type == model.StatusGood {
		w := s.Warnings
		if w == nil {
			w = []string{} // "empty array when none, never omitted"
		}
		d.Warnings = &w
	}
	return d
}

func toStatus(s model.Status) statusDTO {
	return statusDTO{
		statusCustomerDTO: toStatusCustomer(s),
		CreatorID:         s.CreatorID,
		CreationTime:      apitime.Format(s.CreationTime),
	}
}

func toTrackingBase(t model.Tracking) trackingBaseDTO {
	d := trackingBaseDTO{
		TrackingNumber: t.TrackingNumber,
		Cont:           t.Cont,
		Seal:           t.Seal,
		POL:            t.POL,
		POD:            t.POD,
	}
	if p := t.Physical; p != nil {
		d.PhysicalDetails = &physicalDetailsDTO{
			PcsCount: p.PcsCount,
			WeightKg: p.WeightKg,
			Dimensions: dimensionsDTO{
				LengthCm: p.Dimensions.LengthCm,
				WidthCm:  p.Dimensions.WidthCm,
				HeightCm: p.Dimensions.HeightCm,
			},
		}
	}
	return d
}

func toTrackingDetails(t model.Tracking) trackingDetailsDTO {
	return trackingDetailsDTO{
		trackingBaseDTO: toTrackingBase(t),
		CreatorID:       t.CreatorID,
		CreationTime:    apitime.Format(t.CreationTime),
		CustomerID:      t.CustomerID,
	}
}

func toLoginSuccess(a model.Account) loginSuccessResponse {
	return loginSuccessResponse{
		Name:           a.Name,
		AccountCreated: apitime.Format(a.CreateTime),
		Role:           string(a.Role),
	}
}
