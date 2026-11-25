package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/users/internal/api"
	"github.com/AlexeBabanov/sto/users/internal/db"
	"github.com/jackc/pgx/v5"
	openapiTypes "github.com/oapi-codegen/runtime/types"
)

func (h *Handler) GetMyProfile(ctx context.Context, _ api.GetMyProfileRequestObject) (api.GetMyProfileResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	row, err := h.directory.q.GetMyProfile(ctx, user.ID)
	if err != nil {
		return nil, profileReadError(err)
	}
	return api.GetMyProfile200JSONResponse(profileResponse(row)), nil
}

func (h *Handler) UpdateMyNotifications(ctx context.Context, req api.UpdateMyNotificationsRequestObject) (api.UpdateMyNotificationsResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	row, err := h.directory.q.UpdateMyNotifications(ctx, db.UpdateMyNotificationsParams{
		UserID: user.ID, Enabled: req.Body.NotificationsEnabled,
	})
	if err != nil {
		return nil, profileReadError(err)
	}
	return api.UpdateMyNotifications200JSONResponse(profileResponse(db.GetMyProfileRow(row))), nil
}

func profileResponse(row db.GetMyProfileRow) api.MyProfile {
	result := api.MyProfile{
		Name: row.Name, Phone: row.Phone, Role: api.MyProfileRole(row.Role),
		CreatedAt: row.CreatedAt, NotificationsEnabled: row.NotificationsEnabled,
	}
	if row.WorkStartedAt != nil {
		result.WorkStartedAt = &openapiTypes.Date{Time: *row.WorkStartedAt}
	}
	return result
}

func profileReadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.Unauthorized()
	}
	return fmt.Errorf("read own profile: %w", err)
}
