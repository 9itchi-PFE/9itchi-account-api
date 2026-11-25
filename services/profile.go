package services

import (
	"net/url"
	"regexp"
	"strings"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/repositories"
)

var (
	phoneRe    = regexp.MustCompile(`^\+?[0-9]{8,15}$`)
	linkedinRe = regexp.MustCompile(`^/in/([\p{L}\p{N}_%-]{3,100})/?$`)
	githubRe   = regexp.MustCompile(`^/([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?)/?$`)
)

// profilePatch reads the optional fields of the profile from the request: each one trimmed, a phone number without
// its spaces, a LinkedIn or GitHub page reduced to its canonical address. A field left out stays nil (unchanged);
// an empty one clears the value.
func profilePatch(req api.UpdateMeRequest) (repositories.ProfilePatch, *models.ApiError) {
	p := repositories.ProfilePatch{
		School: trimmed(req.School), City: trimmed(req.City), Speciality: trimmed(req.Speciality), Year: trimmed(req.Year),
		Phone: trimmed(req.Phone), LinkedIn: trimmed(req.LinkedIn), GitHub: trimmed(req.GitHub),
	}
	if p.Phone != nil && *p.Phone != "" {
		v := strings.NewReplacer(" ", "", "-", "", ".", "").Replace(*p.Phone)
		if !phoneRe.MatchString(v) {
			return p, badRequest("phone must be 8 to 15 digits, with an optional + in front")
		}
		p.Phone = &v
	}
	if p.LinkedIn != nil && *p.LinkedIn != "" {
		handle, ok := pagePath(*p.LinkedIn, "linkedin.com", linkedinRe)
		if !ok {
			return p, badRequest("linkedin must be the address of a profile, like linkedin.com/in/your-name")
		}
		v := "https://www.linkedin.com/in/" + handle
		p.LinkedIn = &v
	}
	if p.GitHub != nil && *p.GitHub != "" {
		handle, ok := pagePath(*p.GitHub, "github.com", githubRe)
		if !ok {
			return p, badRequest("github must be the address of a profile, like github.com/your-name")
		}
		v := "https://github.com/" + handle
		p.GitHub = &v
	}
	return p, nil
}

func trimmed(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	return &s
}

// pagePath returns the handle of a profile page on site (or one of its subdomains, such as tn.linkedin.com), given
// with or without https://. Any other site, scheme or path is refused.
func pagePath(raw, site string, path *regexp.Regexp) (string, bool) {
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if host != site && !strings.HasSuffix(host, "."+site) {
		return "", false
	}
	m := path.FindStringSubmatch(u.EscapedPath())
	if m == nil {
		return "", false
	}
	return m[1], true
}
