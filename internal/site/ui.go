package site

import "barrypre.com/webcv/internal/data"

// uiText is the site's own furniture — navigation, headings, buttons, form
// labels and status messages. It is deliberately separate from internal/data,
// which holds the CV itself: the CV is what the site is about, and this is the
// page around it.
//
// The fields are resolved into plain strings once per request, so a template
// writes {{.UI.NavHome}} rather than calling a lookup with a string key. A
// typo in a field name then fails to render at startup instead of quietly
// producing an empty element on a page nobody checked.
type uiText struct {
	SkipToContent string
	NavHome       string
	NavExperience string
	NavProjects   string
	NavSkills     string
	NavContact    string
	OpenQuickJump string
	ToggleMenu    string
	ToggleTheme   string
	SwitchTo      string // the label on the other language's link

	PaletteLabel    string
	PalettePages    string
	PaletteRoles    string
	PaletteProjects string
	PaletteNoMatch  string
	PaletteHome     string
	PaletteRoleN    string // "%d roles" — the count is substituted in the template
	PaletteBuildN   string
	PaletteStack    string

	HeroViewWork string
	HeroContact  string
	HeroGetCV    string
	Available    string

	Experience    string
	Projects      string
	Skills        string
	Education     string
	Languages     string
	Contact       string
	Details       string // the timeline expand button
	DetailsFor    string // visually hidden continuation: "details for <company>"
	FilterRole    string // the noun cv-filter counts
	FilterProject string

	ContactEmail    string
	ContactPhone    string
	ContactLinkedIn string
	ContactGitHub   string
	ContactBase     string
	ContactCV       string
	DownloadPDF     string

	FieldName    string
	FieldEmail   string
	FieldMessage string
	SendMessage  string

	StatusSent   string
	StatusBusy   string
	StatusFailed string
	StatusError  string
}

// ui holds every string in both languages. Adding a language means filling in
// a third field here and in internal/data, and nothing else.
var ui = struct {
	SkipToContent data.T
	NavHome       data.T
	NavExperience data.T
	NavProjects   data.T
	NavSkills     data.T
	NavContact    data.T
	OpenQuickJump data.T
	ToggleMenu    data.T
	ToggleTheme   data.T

	PaletteLabel    data.T
	PalettePages    data.T
	PaletteRoles    data.T
	PaletteProjects data.T
	PaletteNoMatch  data.T
	PaletteHome     data.T
	PaletteRoleN    data.T
	PaletteBuildN   data.T
	PaletteStack    data.T

	HeroViewWork data.T
	HeroContact  data.T
	HeroGetCV    data.T
	Available    data.T

	Experience    data.T
	Projects      data.T
	Skills        data.T
	Education     data.T
	Languages     data.T
	Contact       data.T
	Details       data.T
	DetailsFor    data.T
	FilterRole    data.T
	FilterProject data.T

	ContactEmail    data.T
	ContactPhone    data.T
	ContactLinkedIn data.T
	ContactGitHub   data.T
	ContactBase     data.T
	ContactCV       data.T
	DownloadPDF     data.T

	FieldName    data.T
	FieldEmail   data.T
	FieldMessage data.T
	SendMessage  data.T

	StatusSent   data.T
	StatusBusy   data.T
	StatusFailed data.T
	StatusError  data.T
}{
	SkipToContent: data.T{EN: "Skip to content", ES: "Saltar al contenido"},
	NavHome:       data.T{EN: "home", ES: "inicio"},
	NavExperience: data.T{EN: "experience", ES: "experiencia"},
	NavProjects:   data.T{EN: "projects", ES: "proyectos"},
	NavSkills:     data.T{EN: "skills", ES: "competencias"},
	NavContact:    data.T{EN: "contact", ES: "contacto"},
	OpenQuickJump: data.T{EN: "Open quick jump", ES: "Abrir salto rápido"},
	ToggleMenu:    data.T{EN: "Toggle menu", ES: "Abrir o cerrar el menú"},
	ToggleTheme:   data.T{EN: "Toggle theme", ES: "Cambiar el tema"},

	PaletteLabel:    data.T{EN: "Quick jump", ES: "Salto rápido"},
	PalettePages:    data.T{EN: "pages", ES: "páginas"},
	PaletteRoles:    data.T{EN: "roles", ES: "puestos"},
	PaletteProjects: data.T{EN: "projects", ES: "proyectos"},
	PaletteNoMatch:  data.T{EN: "No matches.", ES: "Sin resultados."},
	PaletteHome:     data.T{EN: "the short version", ES: "la versión breve"},
	PaletteRoleN:    data.T{EN: "roles", ES: "puestos"},
	PaletteBuildN:   data.T{EN: "builds", ES: "desarrollos"},
	PaletteStack:    data.T{EN: "stack and education", ES: "stack y formación"},

	HeroViewWork: data.T{EN: "$ view experience", ES: "$ ver experiencia"},
	HeroContact:  data.T{EN: "$ contact --email", ES: "$ contacto --email"},
	HeroGetCV:    data.T{EN: "$ get cv.pdf", ES: "$ descargar cv.pdf"},
	Available:    data.T{EN: "AVAILABLE", ES: "DISPONIBLE"},

	Experience:    data.T{EN: "Experience", ES: "Experiencia"},
	Projects:      data.T{EN: "Projects", ES: "Proyectos"},
	Skills:        data.T{EN: "Skills", ES: "Competencias"},
	Education:     data.T{EN: "education", ES: "formación"},
	Languages:     data.T{EN: "languages", ES: "idiomas"},
	Contact:       data.T{EN: "Contact", ES: "Contacto"},
	Details:       data.T{EN: "+ details", ES: "+ detalles"},
	DetailsFor:    data.T{EN: " for ", ES: " de "},
	FilterRole:    data.T{EN: "role", ES: "puesto"},
	FilterProject: data.T{EN: "project", ES: "proyecto"},

	ContactEmail:    data.T{EN: "email", ES: "correo"},
	ContactPhone:    data.T{EN: "phone", ES: "teléfono"},
	ContactLinkedIn: data.T{EN: "linkedin", ES: "linkedin"},
	ContactGitHub:   data.T{EN: "github", ES: "github"},
	ContactBase:     data.T{EN: "base", ES: "ubicación"},
	ContactCV:       data.T{EN: "cv", ES: "cv"},
	DownloadPDF:     data.T{EN: "download as PDF", ES: "descargar en PDF"},

	FieldName:    data.T{EN: "Name", ES: "Nombre"},
	FieldEmail:   data.T{EN: "Email", ES: "Correo electrónico"},
	FieldMessage: data.T{EN: "Message", ES: "Mensaje"},
	SendMessage:  data.T{EN: "$ send message", ES: "$ enviar mensaje"},

	StatusSent: data.T{
		EN: "Message sent. I will reply soon.",
		ES: "Mensaje enviado. Responderé pronto.",
	},
	StatusBusy: data.T{
		EN: "Too many messages just now. Wait a minute and try again.",
		ES: "Demasiados mensajes por ahora. Espera un minuto e inténtalo de nuevo.",
	},
	StatusFailed: data.T{
		EN: "That did not send — something is wrong on my end, not yours. Please email me directly at",
		ES: "No se ha podido enviar: el fallo está en mi lado, no en el tuyo. Escríbeme directamente a",
	},
	StatusError: data.T{
		EN: "That did not send. Check every field and try again.",
		ES: "No se ha podido enviar. Revisa todos los campos e inténtalo de nuevo.",
	},
}

// textFor resolves every string for one language, once, at startup.
func textFor(l data.Lang) uiText {
	return uiText{
		SkipToContent: ui.SkipToContent.In(l),
		NavHome:       ui.NavHome.In(l),
		NavExperience: ui.NavExperience.In(l),
		NavProjects:   ui.NavProjects.In(l),
		NavSkills:     ui.NavSkills.In(l),
		NavContact:    ui.NavContact.In(l),
		OpenQuickJump: ui.OpenQuickJump.In(l),
		ToggleMenu:    ui.ToggleMenu.In(l),
		ToggleTheme:   ui.ToggleTheme.In(l),
		SwitchTo:      switchLabel(l),

		PaletteLabel:    ui.PaletteLabel.In(l),
		PalettePages:    ui.PalettePages.In(l),
		PaletteRoles:    ui.PaletteRoles.In(l),
		PaletteProjects: ui.PaletteProjects.In(l),
		PaletteNoMatch:  ui.PaletteNoMatch.In(l),
		PaletteHome:     ui.PaletteHome.In(l),
		PaletteRoleN:    ui.PaletteRoleN.In(l),
		PaletteBuildN:   ui.PaletteBuildN.In(l),
		PaletteStack:    ui.PaletteStack.In(l),

		HeroViewWork: ui.HeroViewWork.In(l),
		HeroContact:  ui.HeroContact.In(l),
		HeroGetCV:    ui.HeroGetCV.In(l),
		Available:    ui.Available.In(l),

		Experience:    ui.Experience.In(l),
		Projects:      ui.Projects.In(l),
		Skills:        ui.Skills.In(l),
		Education:     ui.Education.In(l),
		Languages:     ui.Languages.In(l),
		Contact:       ui.Contact.In(l),
		Details:       ui.Details.In(l),
		DetailsFor:    ui.DetailsFor.In(l),
		FilterRole:    ui.FilterRole.In(l),
		FilterProject: ui.FilterProject.In(l),

		ContactEmail:    ui.ContactEmail.In(l),
		ContactPhone:    ui.ContactPhone.In(l),
		ContactLinkedIn: ui.ContactLinkedIn.In(l),
		ContactGitHub:   ui.ContactGitHub.In(l),
		ContactBase:     ui.ContactBase.In(l),
		ContactCV:       ui.ContactCV.In(l),
		DownloadPDF:     ui.DownloadPDF.In(l),

		FieldName:    ui.FieldName.In(l),
		FieldEmail:   ui.FieldEmail.In(l),
		FieldMessage: ui.FieldMessage.In(l),
		SendMessage:  ui.SendMessage.In(l),

		StatusSent:   ui.StatusSent.In(l),
		StatusBusy:   ui.StatusBusy.In(l),
		StatusFailed: ui.StatusFailed.In(l),
		StatusError:  ui.StatusError.In(l),
	}
}

// switchLabel names the other language in its own words, which is the one
// convention a reader who cannot read the current page can still follow.
func switchLabel(current data.Lang) string {
	if current == data.ES {
		return "English"
	}
	return "Español"
}

// otherLang is the language the switcher points at. With two languages it is
// simply the other one.
func otherLang(current data.Lang) data.Lang {
	if current == data.ES {
		return data.EN
	}
	return data.ES
}

// pageTitles and pageDescriptions carry the per-page <title> and meta
// description in both languages. They sit here rather than in internal/data
// because they describe the pages, not the CV.
var pageMeta = map[string]struct{ Title, Description data.T }{
	"/": {
		Title: data.T{
			EN: "Barry Prendergast — Senior Full-Stack Engineer",
			ES: "Barry Prendergast — Ingeniero Full-Stack Senior",
		},
		Description: data.T{
			EN: "Barry Prendergast, Senior Full-Stack Engineer in Madrid. Fifteen years of Angular at enterprise scale, now architecting security-first platforms in Go.",
			ES: "Barry Prendergast, Ingeniero Full-Stack Senior en Madrid. Quince años de Angular a escala empresarial y ahora arquitectura de plataformas centradas en la seguridad con Go.",
		},
	},
	"/experience": {
		Title: data.T{
			EN: "Experience — Barry Prendergast",
			ES: "Experiencia — Barry Prendergast",
		},
		Description: data.T{
			EN: "Eight roles across banking, media and medical platforms, from jQuery layouts at Vocento to principal frontend architect at Quality Compusoft.",
			ES: "Ocho puestos en banca, medios de comunicación y plataformas sanitarias, desde maquetación con jQuery en Vocento hasta arquitecto frontend principal en Quality Compusoft.",
		},
	},
	"/projects": {
		Title: data.T{
			EN: "Projects — Barry Prendergast",
			ES: "Proyectos — Barry Prendergast",
		},
		Description: data.T{
			EN: "Selected work: SAUI, a server-authoritative web architecture in Go and htmx, and a GDPR-compliant medical-device ordering portal for Archway Orthotics.",
			ES: "Trabajos destacados: SAUI, una arquitectura web con autoridad en el servidor en Go y htmx, y un portal de pedidos de productos sanitarios conforme al RGPD para Archway Orthotics.",
		},
	},
	"/skills": {
		Title: data.T{
			EN: "Skills — Barry Prendergast",
			ES: "Competencias — Barry Prendergast",
		},
		Description: data.T{
			EN: "Angular v1–20, TypeScript, RxJS and Web Components on the front end; Go, SQLite and OWASP Top 10 security practice on the back.",
			ES: "Angular v1–20, TypeScript, RxJS y Web Components en el frontend; Go, SQLite y prácticas de seguridad OWASP Top 10 en el backend.",
		},
	},
	"/contact": {
		Title: data.T{
			EN: "Contact — Barry Prendergast",
			ES: "Contacto — Barry Prendergast",
		},
		Description: data.T{
			EN: "Email, phone and LinkedIn for Barry Prendergast, or send a message straight from the page. Based in Madrid, Spain.",
			ES: "Correo, teléfono y LinkedIn de Barry Prendergast, o envía un mensaje desde la propia página. Con base en Madrid, España.",
		},
	},
}
