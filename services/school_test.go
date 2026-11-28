package services

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"9itchi-account-api/models/api"
	"9itchi-account-api/models/db"
	"9itchi-account-api/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memSchools keeps schools in memory, in the order they were added.
type memSchools struct {
	items []db.School
	err   error
}

func (m *memSchools) find(code string) int {
	for i := range m.items {
		if m.items[i].Code == code {
			return i
		}
	}
	return -1
}

func (m *memSchools) List(context.Context) ([]db.School, error) { return m.items, m.err }

func (m *memSchools) Create(_ context.Context, s *db.School) (*db.School, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.find(s.Code) >= 0 {
		return nil, repositories.ErrConflict
	}
	m.items = append(m.items, *s)
	return s, nil
}

func (m *memSchools) Update(_ context.Context, s *db.School) (*db.School, error) {
	if m.err != nil {
		return nil, m.err
	}
	i := m.find(s.Code)
	if i < 0 {
		return nil, repositories.ErrNotFound
	}
	m.items[i].NameFR, m.items[i].NameEN, m.items[i].NameAR = s.NameFR, s.NameEN, s.NameAR
	return &m.items[i], nil
}

func (m *memSchools) Delete(_ context.Context, code string) error {
	if m.err != nil {
		return m.err
	}
	i := m.find(code)
	if i < 0 {
		return repositories.ErrNotFound
	}
	m.items = append(m.items[:i], m.items[i+1:]...)
	return nil
}

func TestSchools(t *testing.T) {
	mem := &memSchools{}
	svc := NewSchoolService(mem)
	ctx := context.Background()

	// An empty list is an empty array, not null.
	list, apiErr := svc.List(ctx)
	require.Nil(t, apiErr)
	assert.NotNil(t, list.Items)
	assert.Empty(t, list.Items)

	// The code is trimmed and upper-cased, the names are trimmed.
	res, apiErr := svc.Create(ctx, api.CreateSchoolRequest{Code: " enetcom ", NameFR: " ENET’Com FR ", NameEN: "ENET’Com", NameAR: "ENET’Com AR"})
	require.Nil(t, apiErr)
	assert.Equal(t, api.SchoolResponse{Code: "ENETCOM", NameFR: "ENET’Com FR", NameEN: "ENET’Com", NameAR: "ENET’Com AR"}, *res)

	_, apiErr = svc.Create(ctx, api.CreateSchoolRequest{Code: "ENETCOM", NameFR: "x", NameEN: "x", NameAR: "x"})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Code, "the code is unique")

	for name, req := range map[string]api.CreateSchoolRequest{
		"code too short":  {Code: "A", NameFR: "x", NameEN: "x", NameAR: "x"},
		"code with space": {Code: "EN IS", NameFR: "x", NameEN: "x", NameAR: "x"},
		"code too long":   {Code: strings.Repeat("A", 21), NameFR: "x", NameEN: "x", NameAR: "x"},
		"blank name":      {Code: "OK", NameFR: "  ", NameEN: "x", NameAR: "x"},
		"name too long":   {Code: "OK", NameFR: "x", NameEN: strings.Repeat("é", 161), NameAR: "x"},
	} {
		_, apiErr = svc.Create(ctx, req)
		require.NotNil(t, apiErr, name)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code, name)
	}

	// Rename by code (any case); the code itself stays.
	res, apiErr = svc.Update(ctx, "enetcom", api.UpdateSchoolRequest{NameFR: "Nouveau", NameEN: "New", NameAR: "جديد"})
	require.Nil(t, apiErr)
	assert.Equal(t, api.SchoolResponse{Code: "ENETCOM", NameFR: "Nouveau", NameEN: "New", NameAR: "جديد"}, *res)
	_, apiErr = svc.Update(ctx, "NOPE", api.UpdateSchoolRequest{NameFR: "x", NameEN: "x", NameAR: "x"})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)
	_, apiErr = svc.Update(ctx, "ENETCOM", api.UpdateSchoolRequest{NameFR: "", NameEN: "x", NameAR: "x"})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)

	list, apiErr = svc.List(ctx)
	require.Nil(t, apiErr)
	assert.Equal(t, []api.SchoolResponse{*res}, list.Items)

	require.Nil(t, svc.Delete(ctx, " enetcom "))
	apiErr = svc.Delete(ctx, "ENETCOM")
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)

	// A storage failure is a generic 500.
	mem.err = assert.AnError
	_, apiErr = svc.List(ctx)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.NotContains(t, apiErr.Error.Error(), assert.AnError.Error())
}
