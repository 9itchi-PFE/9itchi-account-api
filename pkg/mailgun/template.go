package mailgun

import (
	"html"
	"strconv"
	"strings"
)

// The two account emails: both carry one 6-digit code.
const (
	kindReset  = "reset"
	kindVerify = "verify"
)

// codeWords are the words of one code email in one language; {n} is the lifetime of the code in minutes.
type codeWords struct {
	Category, Subject, Heading, Text, Expires, Ignore string
}

var codeCatalog = map[string]map[string]codeWords{
	kindVerify: {
		"fr": {"Inscription", "Ton code de vérification 9itchi", "Confirme ton adresse email",
			"Saisis ce code sur 9itchi pour activer ton compte :", "Il expire dans {n} minutes.",
			"Si tu n’as pas créé de compte 9itchi, ignore cet email."},
		"en": {"Sign-up", "Your 9itchi verification code", "Confirm your email address",
			"Enter this code on 9itchi to activate your account:", "It expires in {n} minutes.",
			"If you did not create a 9itchi account, you can ignore this email."},
		"ar": {"التسجيل", "رمز التحقق الخاص بك على 9itchi", "أكّد بريدك الإلكتروني",
			"أدخل هذا الرمز على 9itchi لتفعيل حسابك:", "تنتهي صلاحيته خلال {n} دقيقة.",
			"إن لم تنشئ حسابًا على 9itchi، تجاهل هذه الرسالة."},
	},
	kindReset: {
		"fr": {"Sécurité", "Ton code de réinitialisation 9itchi", "Réinitialise ton mot de passe",
			"Saisis ce code sur 9itchi pour choisir un nouveau mot de passe :", "Il expire dans {n} minutes.",
			"Si tu n’as rien demandé, ignore cet email : ton mot de passe ne change pas."},
		"en": {"Security", "Your 9itchi password reset code", "Reset your password",
			"Enter this code on 9itchi to choose a new password:", "It expires in {n} minutes.",
			"If you did not ask for this, ignore this email: your password stays the same."},
		"ar": {"الأمان", "رمز إعادة تعيين كلمة المرور على 9itchi", "أعد تعيين كلمة المرور",
			"أدخل هذا الرمز على 9itchi لاختيار كلمة مرور جديدة:", "تنتهي صلاحيته خلال {n} دقيقة.",
			"إن لم تطلب ذلك، تجاهل هذه الرسالة: كلمة مرورك لن تتغيّر."},
	},
}

var codeFooters = map[string]string{
	"fr": "Tu reçois cet email car ton adresse a été utilisée sur 9itchi.",
	"en": "You receive this email because your address was used on 9itchi.",
	"ar": "تصلك هذه الرسالة لأن عنوانك استُخدم على 9itchi.",
}

// Fonts of the app (client tailwind.config.ts): Bricolage Grotesque headings, Inter text, Noto Sans Arabic, JetBrains
// Mono for the code. Mail clients that load no web font fall back to Arial and Courier.
const (
	fontText    = "Inter,'Noto Sans Arabic',Arial,Helvetica,sans-serif"
	fontDisplay = "'Bricolage Grotesque',Inter,'Noto Sans Arabic',Arial,Helvetica,sans-serif"
	fontMono    = "'JetBrains Mono','Courier New',Courier,monospace"
	fontsCSS    = "https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:wght@800&family=Inter:wght@400;600&family=JetBrains+Mono:wght@700&family=Noto+Sans+Arabic:wght@400;600&display=swap"
)

// renderCode writes the subject, plain-text and HTML bodies of a code email. ok is false for an unknown kind.
func renderCode(kind, lang, code string, minutes int) (subject, text, htmlBody string, ok bool) {
	if _, known := codeFooters[lang]; !known {
		lang = "fr"
	}
	w, ok := codeCatalog[kind][lang]
	if !ok {
		return "", "", "", false
	}
	expires := strings.ReplaceAll(w.Expires, "{n}", strconv.Itoa(minutes))
	footer := codeFooters[lang]
	text = w.Heading + "\n\n" + w.Text + "\n\n" + code + "\n\n" + expires + "\n" + w.Ignore + "\n\n" + footer + "\n"

	dir, align, opposite := "ltr", "left", "right"
	if lang == "ar" {
		dir, align, opposite = "rtl", "right", "left"
	}
	e := html.EscapeString
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="` + lang + `" dir="` + dir + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>` + e(w.Subject) + `</title><link rel="stylesheet" href="` + fontsCSS + `"></head>`)
	b.WriteString(`<body style="margin:0;padding:0;background:#FAFAF9;">`)
	b.WriteString(`<div style="display:none;max-height:0;overflow:hidden;opacity:0;">` + e(w.Text) + ` ` + e(code) + `</div>`)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#FAFAF9;"><tr><td align="center" style="padding:24px 12px;">`)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" dir="` + dir + `" style="max-width:520px;background:#ffffff;border:1.5px solid #101218;border-radius:18px;box-shadow:5px 5px 0 #101218;font-family:` + fontText + `;font-size:14px;line-height:1.6;color:#344054;text-align:` + align + `;"><tr><td style="padding:24px;">`)
	// Logo and category.
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr>`)
	b.WriteString(`<td style="text-align:` + align + `;"><span style="display:inline-block;width:22px;height:22px;line-height:22px;border-radius:6px;background:#D6F25E;text-align:center;font-weight:800;font-size:13px;color:#101218;font-family:` + fontDisplay + `;">9</span> <strong style="color:#101218;font-family:` + fontDisplay + `;font-size:15px;">9itchi</strong></td>`)
	b.WriteString(`<td style="text-align:` + opposite + `;"><span style="display:inline-block;height:24px;line-height:24px;padding:0 9px;border-radius:999px;background:#F2F3F5;color:#344054;font-size:12px;font-weight:600;">` + e(w.Category) + `</span></td>`)
	b.WriteString(`</tr></table>`)
	b.WriteString(`<h1 style="margin:18px 0 0;font-family:` + fontDisplay + `;font-size:21px;font-weight:800;line-height:1.3;letter-spacing:-0.01em;color:#101218;">` + e(w.Heading) + `</h1>`)
	b.WriteString(`<p style="margin:12px 0 0;">` + e(w.Text) + `</p>`)
	// The code reads left to right in every language.
	b.WriteString(`<p dir="ltr" style="margin:16px 0 0;text-align:center;"><span style="display:inline-block;padding:12px 22px;border:1.5px solid #101218;border-radius:12px;background:#D6F25E;font-family:` + fontMono + `;font-size:30px;font-weight:700;letter-spacing:8px;color:#101218;">` + e(code) + `</span></p>`)
	b.WriteString(`<p style="margin:16px 0 0;font-weight:600;color:#101218;">` + e(expires) + `</p>`)
	b.WriteString(`<p style="margin:4px 0 0;color:#596070;">` + e(w.Ignore) + `</p>`)
	b.WriteString(`<p style="margin:16px 0 0;font-size:11.5px;color:#98A2B3;">` + e(footer) + `</p>`)
	b.WriteString(`</td></tr></table></td></tr></table></body></html>`)
	return w.Subject, text, b.String(), true
}
