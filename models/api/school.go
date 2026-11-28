package api

// SchoolResponse is one school of the list offered to students, named in each language of the app.
type SchoolResponse struct {
	Code   string `json:"code"`
	NameFR string `json:"name_fr"`
	NameEN string `json:"name_en"`
	NameAR string `json:"name_ar"`
}

// ListSchoolsResponse is the whole list of schools, in the order shown to students.
type ListSchoolsResponse struct {
	Items []SchoolResponse `json:"items"`
}

// CreateSchoolRequest is the body of CreateSchool. Code is the short code kept in users.school (A-Z, 0-9, _ and -,
// 2 to 20 characters); it is upper-cased and cannot be changed afterwards.
type CreateSchoolRequest struct {
	Code   string `json:"code" binding:"required"`
	NameFR string `json:"name_fr" binding:"required"`
	NameEN string `json:"name_en" binding:"required"`
	NameAR string `json:"name_ar" binding:"required"`
}

// UpdateSchoolRequest is the body of UpdateSchool: the names of the school, all three are replaced.
type UpdateSchoolRequest struct {
	NameFR string `json:"name_fr" binding:"required"`
	NameEN string `json:"name_en" binding:"required"`
	NameAR string `json:"name_ar" binding:"required"`
}
