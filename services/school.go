package services

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/models/db"
	"9itchi-account-api/repositories"
)

const maxSchoolNameLength = 160

var schoolCodeRe = regexp.MustCompile(`^[A-Z0-9_-]{2,20}$`)

// SchoolService is the list of schools students pick from at sign-up and in settings. Anyone can read it; the admins
// add, rename and remove schools. A removed school leaves the users who chose it untouched (their value stays).
type SchoolService interface {
	// GET /schools (public): every school, in the order shown to students
	List(ctx context.Context) (*api.ListSchoolsResponse, *models.ApiError)
	// POST /admin/schools: add a school (409 when the code is taken)
	Create(ctx context.Context, req api.CreateSchoolRequest) (*api.SchoolResponse, *models.ApiError)
	// PUT /admin/schools/:code: rename a school (the code never changes)
	Update(ctx context.Context, code string, req api.UpdateSchoolRequest) (*api.SchoolResponse, *models.ApiError)
	// DELETE /admin/schools/:code: remove a school from the list
	Delete(ctx context.Context, code string) *models.ApiError
}

type schoolService struct {
	schools repositories.SchoolRepository
}

func NewSchoolService(schools repositories.SchoolRepository) SchoolService {
	return &schoolService{schools: schools}
}

func (s *schoolService) List(ctx context.Context) (*api.ListSchoolsResponse, *models.ApiError) {
	items, err := s.schools.List(ctx)
	if err != nil {
		return nil, internalError("list schools", err)
	}
	res := &api.ListSchoolsResponse{Items: make([]api.SchoolResponse, 0, len(items))}
	for i := range items {
		res.Items = append(res.Items, schoolView(&items[i]))
	}
	return res, nil
}

func (s *schoolService) Create(ctx context.Context, req api.CreateSchoolRequest) (*api.SchoolResponse, *models.ApiError) {
	code := normalizeSchoolCode(req.Code)
	if !schoolCodeRe.MatchString(code) {
		return nil, badRequest("code must be 2 to 20 characters: letters, digits, _ and -")
	}
	names, apiErr := cleanSchoolNames(req.NameFR, req.NameEN, req.NameAR)
	if apiErr != nil {
		return nil, apiErr
	}
	created, err := s.schools.Create(ctx, &db.School{Code: code, NameFR: names[0], NameEN: names[1], NameAR: names[2]})
	if err != nil {
		return nil, schoolError("create school", err)
	}
	v := schoolView(created)
	return &v, nil
}

func (s *schoolService) Update(ctx context.Context, code string, req api.UpdateSchoolRequest) (*api.SchoolResponse, *models.ApiError) {
	names, apiErr := cleanSchoolNames(req.NameFR, req.NameEN, req.NameAR)
	if apiErr != nil {
		return nil, apiErr
	}
	updated, err := s.schools.Update(ctx, &db.School{Code: normalizeSchoolCode(code), NameFR: names[0], NameEN: names[1], NameAR: names[2]})
	if err != nil {
		return nil, schoolError("update school", err)
	}
	v := schoolView(updated)
	return &v, nil
}

func (s *schoolService) Delete(ctx context.Context, code string) *models.ApiError {
	if err := s.schools.Delete(ctx, normalizeSchoolCode(code)); err != nil {
		return schoolError("delete school", err)
	}
	return nil
}

func normalizeSchoolCode(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

// cleanSchoolNames trims the names (fr, en, ar) and refuses an empty or too long one.
func cleanSchoolNames(names ...string) ([3]string, *models.ApiError) {
	var out [3]string
	for i, n := range names {
		n = strings.TrimSpace(n)
		if n == "" || utf8.RuneCountInString(n) > maxSchoolNameLength {
			return out, badRequest("each name must be 1 to 160 characters")
		}
		out[i] = n
	}
	return out, nil
}

func schoolView(s *db.School) api.SchoolResponse {
	return api.SchoolResponse{Code: s.Code, NameFR: s.NameFR, NameEN: s.NameEN, NameAR: s.NameAR}
}

func schoolError(op string, err error) *models.ApiError {
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		return apiError(http.StatusNotFound, "school not found")
	case errors.Is(err, repositories.ErrConflict):
		return apiError(http.StatusConflict, "a school with this code already exists")
	}
	return internalError(op, err)
}
