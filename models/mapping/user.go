package mapping

import (
	"9itchi-account-api/models/api"
	"9itchi-account-api/models/db"
)

func UserToResponse(u *db.User) api.UserResponse {
	res := api.UserResponse{
		ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, Status: u.Status, Language: u.Language,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
	res.School, res.Phone, res.City = str(u.School), str(u.Phone), str(u.City)
	res.Speciality, res.Year, res.LinkedIn, res.GitHub = str(u.Speciality), str(u.Year), str(u.LinkedIn), str(u.GitHub)
	res.HasPhoto = u.HasPhoto
	return res
}

func UserToInternal(u *db.User) api.InternalUserResponse {
	return api.InternalUserResponse{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, Status: u.Status, Language: u.Language}
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
