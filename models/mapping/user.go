package mapping

import (
	"9itchi-account-api/models/api"
	"9itchi-account-api/models/db"
)

func UserToResponse(u *db.User) api.UserResponse {
	res := api.UserResponse{
		ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, Status: u.Status,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
	if u.School != nil {
		res.School = *u.School
	}
	return res
}

func UserToInternal(u *db.User) api.InternalUserResponse {
	return api.InternalUserResponse{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, Status: u.Status}
}
