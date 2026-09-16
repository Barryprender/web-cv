// Package data holds Barry Prendergast's CV content as plain Go values.
// It is the single source of truth the site templates render from.
//
// Every piece of prose is a T, holding the English and the Spanish in one
// literal. See lang.go for why they sit together rather than in parallel files.
package data

// Contact holds how to reach Barry.
type Contact struct {
	Email    string
	Phone    string
	LinkedIn string
	GitHub   string
	Site     string
	Location T
}

// Link is a labelled destination. Jobs can carry several: the Secure UI
// Components work lives at a showcase site, a repository and an npm package,
// and a single URL field could only point at one of them.
//
// Label is not a T: these are domain names and package names, which are the
// same in every language.
type Link struct {
	Label string
	URL   string
}

// Language is a spoken language and the level claimed for it.
type Language struct {
	Name  T
	Level T
}

// Job is one entry in the experience timeline.
type Job struct {
	// Company is a proper noun for a real employer, so it is not translated.
	// The entries that are not employers — a career break, a period of study —
	// are descriptions rather than names, and those are.
	Company  T
	Title    T
	From     Date
	To       Date // zero for the current role
	Location T
	Summary  T // optional one-line context; empty if not needed
	Bullets  TS
	Stack    []string // technology names, the same in both languages
	Links    []Link   // where the work can actually be seen, if anywhere

	// Kind separates an employer relationship from the entries that are not
	// one. Empty means employment, so every real role is unchanged by this
	// field existing. Anything else must never be rendered, or described in
	// structured data, as a job.
	Kind string // "", "independent", "break"
}

// IsEmployment reports whether this entry represents working for someone else.
func (j Job) IsEmployment() bool { return j.Kind == "" }

// Period writes the role's span in l. The current role has a zero To, which
// Date renders as "Present" or its translation.
func (j Job) Period(l Lang) string { return Range(j.From, j.To, l) }

// Project is one piece of work on the projects page. Unlike a Job it may have
// no employer and no dates — what matters is what it is and where to see it.
type Project struct {
	Name     string // the project's own name, not translated
	Role     T      // "Own project", "Client engagement", "Product demo"
	From, To Date   // both zero when the work is ongoing or undated
	Links    []Link // public destinations, empty when there is nothing to show
	Summary  T
	Bullets  TS
	Stack    []string
}

// Dated reports whether this project has a period worth printing.
func (p Project) Dated() bool { return p.From.Year != 0 }

// Period writes the project's span in l, or "" when it has none.
func (p Project) Period(l Lang) string {
	if !p.Dated() {
		return ""
	}
	return Range(p.From, p.To, l)
}

// SkillGroup is a named cluster of skills shown together.
type SkillGroup struct {
	Category T
	Skills   TS
}

// EducationEntry is one line of education or certification.
type EducationEntry struct {
	Institution string // the school's name, not translated
	Program     T
	From, To    Date
	Detail      TS // syllabus, final project, anything worth naming
}

// Me is the single source of truth for the site's content.
var Me = struct {
	Name     string
	Headline T
	Tagline  T
	// Status is the line on the home page saying what is in flight. It lives
	// here rather than in the template because it has to stay consistent with
	// Jobs and Projects, and copy kept in a template drifts out of step with
	// the data it describes.
	Status    T
	Contact   Contact
	Jobs      []Job
	Projects  []Project
	Skills    []SkillGroup
	Education []EducationEntry
	Languages []Language
}{
	Name: "Barry Prendergast",
	Headline: T{
		EN: "Senior Full-Stack Engineer",
		ES: "Ingeniero Full-Stack Senior",
	},
	Tagline: T{
		EN: "Fifteen years of Angular at enterprise scale. Now building security-first platforms in Go.",
		ES: "Quince años de Angular a escala empresarial. Ahora construyo plataformas centradas en la seguridad con Go.",
	},
	Status: T{
		EN: "building saui and secure-ui-components · shipped the archway orthotics portal",
		ES: "desarrollando saui y secure-ui-components · portal de archway orthotics entregado",
	},
	Contact: Contact{
		Email:    "barryprendergast78@gmail.com",
		Phone:    "+34 667 454 227",
		LinkedIn: "https://www.linkedin.com/in/barrypdrgst",
		GitHub:   "https://github.com/Barryprender",
		// The canonical origin. Everything derives from this: rel=canonical,
		// og:url, the sitemap, the JSON-LD, the address printed on the PDF and
		// the footer of every contact email. Change it here and nowhere else.
		//
		// Currently the Fly hostname because barrypre.com is not registered —
		// pointing canonical URLs at a domain that does not resolve is worse
		// than an unlovely hostname that does.
		Site:     "https://barrypre-web.fly.dev",
		Location: T{EN: "Madrid, Spain", ES: "Madrid, España"},
	},
	Jobs: []Job{
		{
			Company: T{EN: "Independent", ES: "Independiente"},
			Title: T{
				EN: "Open source, client engagements and cybersecurity training",
				ES: "Código abierto, proyectos para clientes y formación en ciberseguridad",
			},
			Kind:     "independent",
			From:     Date{Year: 2026, Month: 3},
			Location: T{EN: "Madrid", ES: "Madrid"},
			Bullets: TS{
				EN: []string{
					"Built and published Secure UI Components and SAUI, and delivered the Archway Orthotics practitioner portal as a client engagement (see Projects)",
					"Completed the Advanced Cybersecurity & Cyber Intelligence diploma at CADEL, covering penetration testing methodology, SOC analysis, and the Kali Linux, Burp Suite, OWASP ZAP, SQLMap and Metasploit toolchain",
				},
				ES: []string{
					"Desarrollé y publiqué Secure UI Components y SAUI, y entregué el portal para profesionales de Archway Orthotics como proyecto para cliente (ver Proyectos)",
					"Completé el diploma en Ciberseguridad Avanzada e Inteligencia Cibernética en CADEL, que abarca metodología de test de intrusión, análisis SOC y las herramientas Kali Linux, Burp Suite, OWASP ZAP, SQLMap y Metasploit",
				},
			},
			Stack: []string{"Go", "TypeScript", "Web Components", "SQLite", "Fly.io"},
		},
		{
			Company: T{EN: "Quality Compusoft", ES: "Quality Compusoft"},
			Title: T{
				EN: "Senior Frontend Developer",
				ES: "Desarrollador Frontend Senior",
			},
			From:     Date{Year: 2020, Month: 9},
			To:       Date{Year: 2026, Month: 3},
			Location: T{EN: "Madrid", ES: "Madrid"},
			Summary: T{
				EN: "Principal frontend architect for Telefy, an enterprise media management platform for dental and veterinary TV networks.",
				ES: "Arquitecto frontend principal de Telefy, una plataforma empresarial de gestión de medios para redes de televisión de clínicas dentales y veterinarias.",
			},
			Bullets: TS{
				EN: []string{
					"Architected a complete Angular application from the ground up (v13 → v18 → v20): a dual-interface platform with an admin dashboard and a user-facing app covering device management, real-time streaming, drag-and-drop playlist creation, and social media integrations",
					"Created 115+ custom components, including a 27-component UI library built without external frameworks (inputs, selects, dropzones, modals, sliders, media viewers), with hand-coded responsive CSS on Grid/Flexbox, cutting bundle size 40%",
					"Implemented the frontend UI and user flows for social integrations (Facebook, Instagram, TikTok, YouTube, X/Twitter): OAuth authorization screens, connection status, content feed views, and posting workflows, in collaboration with the backend team",
					"Shipped i18n across 4 languages (Spanish, English, Catalan, Portuguese) with dynamic switching, a real-time streaming interface, and remote device configuration across multiple TV devices",
					"Led a refactor removing layout wrappers from 50+ components and upgraded the legacy app from Angular v12 to v17 (dependency-limited) while evolving Telefy itself through v13 to v20",
				},
				ES: []string{
					"Diseñé la arquitectura de una aplicación Angular completa desde cero (v13 → v18 → v20): una plataforma de doble interfaz con panel de administración y aplicación de usuario que cubre gestión de dispositivos, streaming en tiempo real, creación de listas de reproducción con arrastrar y soltar, e integraciones con redes sociales",
					"Creé más de 115 componentes propios, incluida una biblioteca de interfaz de 27 componentes construida sin frameworks externos (campos de texto, selectores, zonas de carga, modales, deslizadores, visores multimedia), con CSS responsivo escrito a mano sobre Grid y Flexbox, reduciendo el tamaño del paquete un 40 %",
					"Implementé la interfaz y los flujos de usuario de las integraciones con redes sociales (Facebook, Instagram, TikTok, YouTube, X/Twitter): pantallas de autorización OAuth, estado de conexión, vistas de contenido y flujos de publicación, en colaboración con el equipo de backend",
					"Entregué la internacionalización en 4 idiomas (español, inglés, catalán y portugués) con cambio dinámico, una interfaz de streaming en tiempo real y la configuración remota de múltiples dispositivos de televisión",
					"Dirigí una refactorización que eliminó envoltorios de maquetación de más de 50 componentes y actualicé la aplicación heredada de Angular v12 a v17 (limitada por dependencias), mientras hacía evolucionar Telefy de v13 a v20",
				},
			},
			Stack: []string{"Angular 13-20", "TypeScript", "RxJS", "CSS Grid/Flexbox", "REST APIs"},
		},
		{
			Company:  T{EN: "Career break", ES: "Excedencia"},
			Title:    T{EN: "Parental", ES: "Parental"},
			Kind:     "break",
			From:     Date{Year: 2019, Month: 7},
			To:       Date{Year: 2020, Month: 9},
			Location: T{EN: "Madrid", ES: "Madrid"},
			Summary: T{
				EN: "Full-time carer for two young children.",
				ES: "Cuidado a tiempo completo de dos hijos pequeños.",
			},
		},
		{
			Company: T{EN: "Arcmedia AG", ES: "Arcmedia AG"},
			Title: T{
				EN: "Senior Front-End Developer",
				ES: "Desarrollador Front-End Senior",
			},
			From:     Date{Year: 2017, Month: 7},
			To:       Date{Year: 2019, Month: 7},
			Location: T{EN: "Madrid", ES: "Madrid"},
			Bullets: TS{
				EN: []string{
					"Delivered responsive layouts across client projects, mainly in Drupal, with one project in Angular 8",
					"Configured and implemented email templates with Foundation for Emails",
				},
				ES: []string{
					"Entregué maquetaciones responsivas en proyectos de clientes, principalmente en Drupal, y un proyecto en Angular 8",
					"Configuré e implementé plantillas de correo electrónico con Foundation for Emails",
				},
			},
			Stack: []string{"Drupal", "Angular 8", "Foundation"},
		},
		{
			Company: T{EN: "GRADDO", ES: "GRADDO"},
			Title: T{
				EN: "Senior Frontend Developer",
				ES: "Desarrollador Frontend Senior",
			},
			From:     Date{Year: 2016, Month: 2},
			To:       Date{Year: 2017, Month: 6},
			Location: T{EN: "Madrid", ES: "Madrid"},
			Bullets: TS{
				EN: []string{
					"Built the UI layer for in-house .NET applications with Bootstrap",
					"Worked on a DevOps team building a banking app in Angular 1.5.8",
				},
				ES: []string{
					"Construí la capa de interfaz de aplicaciones .NET internas con Bootstrap",
					"Trabajé en un equipo DevOps desarrollando una aplicación bancaria en Angular 1.5.8",
				},
			},
			Stack: []string{".NET", "Bootstrap", "Angular 1.5"},
		},
		{
			Company: T{EN: "Cecabank", ES: "Cecabank"},
			Title: T{
				EN: "Front-End Developer",
				ES: "Desarrollador Front-End",
			},
			From:     Date{Year: 2015, Month: 4},
			To:       Date{Year: 2016, Month: 2},
			Location: T{EN: "Madrid", ES: "Madrid"},
			Bullets: TS{
				EN: []string{
					"Built and maintained layouts for banking-sector projects in HTML, CSS, and JavaScript",
				},
				ES: []string{
					"Desarrollé y mantuve maquetaciones para proyectos del sector bancario en HTML, CSS y JavaScript",
				},
			},
			Stack: []string{"HTML", "CSS", "JavaScript", "Bootstrap", ".NET"},
		},
		{
			Company: T{EN: "Vivocom", ES: "Vivocom"},
			Title: T{
				EN: "Front-End Developer",
				ES: "Desarrollador Front-End",
			},
			From:     Date{Year: 2014, Month: 9},
			To:       Date{Year: 2015, Month: 2},
			Location: T{EN: "Madrid", ES: "Madrid"},
			Bullets: TS{
				EN: []string{
					"Maquetación web for Vivocom Keepunto: converted PSD designs into responsive HTML, CSS/LESS, Bootstrap and jQuery for banking-sector apps and promotional sites",
				},
				ES: []string{
					"Maquetación web para Vivocom Keepunto: conversión de diseños PSD en HTML responsivo, CSS/LESS, Bootstrap y jQuery para aplicaciones del sector bancario y sitios promocionales",
				},
			},
			Stack: []string{"HTML", "CSS", "LESS", "Bootstrap", "jQuery"},
		},
		{
			Company: T{EN: "Vocento", ES: "Vocento"},
			Title: T{
				EN: "Front-End Developer",
				ES: "Desarrollador Front-End",
			},
			From:     Date{Year: 2013, Month: 4},
			To:       Date{Year: 2014, Month: 9},
			Location: T{EN: "Madrid", ES: "Madrid"},
			Bullets: TS{
				EN: []string{
					"Built responsive, mobile-first web layouts with HTML5, CSS3, and jQuery for digital media properties at one of Spain's largest media groups, including GUAPABOX (fashion e-commerce) and ABC FOTO (photo platform)",
					"Produced responsive templates with Skeleton Grid and Bootstrap, plus landing-page templates for several products",
				},
				ES: []string{
					"Desarrollé maquetaciones web responsivas con enfoque mobile-first en HTML5, CSS3 y jQuery para medios digitales de uno de los mayores grupos de comunicación de España, incluidos GUAPABOX (comercio electrónico de moda) y ABC FOTO (plataforma fotográfica)",
					"Produje plantillas responsivas con Skeleton Grid y Bootstrap, además de plantillas de páginas de aterrizaje para varios productos",
				},
			},
			Stack: []string{"HTML5", "CSS3", "jQuery", "Bootstrap"},
		},
		{
			Company: T{EN: "Full-time study", ES: "Formación a tiempo completo"},
			Title: T{
				EN: "Web Application Development, Ondas Formación",
				ES: "Desarrollo de Aplicaciones Web, Ondas Formación",
			},
			Kind:     "break",
			From:     Date{Year: 2012, Month: 9},
			To:       Date{Year: 2012, Month: 12},
			Location: T{EN: "Madrid", ES: "Madrid"},
			Summary: T{
				EN: "Retrained on HTML5, CSS3, JavaScript and jQuery (see Education).",
				ES: "Reciclaje profesional en HTML5, CSS3, JavaScript y jQuery (ver Formación).",
			},
		},
		{
			Company: T{EN: "Mercantis", ES: "Mercantis"},
			Title: T{
				EN: "Web Layout Developer",
				ES: "Maquetador Web",
			},
			From: Date{Year: 2008},
			To:   Date{Year: 2011},
			Bullets: TS{
				EN: []string{
					"Built web layouts and front-end for health-sector products, and maintained the WordPress sites they ran on",
				},
				ES: []string{
					"Desarrollé maquetación web y front-end para productos del sector sanitario, y mantuve los sitios WordPress sobre los que funcionaban",
				},
			},
			Stack: []string{"HTML", "CSS", "jQuery", "WordPress"},
		},
	},
	Projects: []Project{
		{
			Name: "Secure UI Components",
			Role: T{EN: "Own project", ES: "Proyecto propio"},
			Summary: T{
				EN: "Zero-dependency, security-first Web Components library and its Go-built showcase site.",
				ES: "Biblioteca de Web Components sin dependencias y centrada en la seguridad, con su sitio de demostración construido en Go.",
			},
			Links: []Link{
				{Label: "secure-ui-web.fly.dev", URL: "https://secure-ui-web.fly.dev"},
				{Label: "github.com/Barryprender/Secure-UI", URL: "https://github.com/Barryprender/Secure-UI"},
				{Label: "npm: secure-ui-components", URL: "https://www.npmjs.com/package/secure-ui-components"},
			},
			Bullets: TS{
				EN: []string{
					"Designed and built a zero-runtime-dependency TypeScript Web Components library covering OWASP Top 10:2021 (A01–A10) with security active by default: CSRF protection, XSS sanitisation, audit logging, and closed Shadow DOM isolation, none of it requiring developer configuration",
					"Built <secure-telemetry-provider>: CAPTCHA-free bot detection using HMAC-SHA-256 signed behavioural signal envelopes (webdriver flag, headless detection, mouse/keyboard presence, submit timing, screen dimensions). Signals are verified server-side, and the data never leaves the developer's infrastructure",
					"9 production-ready components published to npm: secure-form, secure-input, secure-select, secure-textarea, secure-file-upload, secure-datetime, secure-table, secure-card, secure-telemetry-provider",
					"Framework-agnostic via the W3C Custom Elements standard: works in Angular, React, Vue, Svelte, Go/templ, Django, Rails, and plain HTML with a single script tag",
					"Shipped the showcase site itself on the Go standard library, SQLite, templ, and native HTML/CSS/TypeScript, deployed to Fly.io with full i18n (EN/ES/FR/DE)",
				},
				ES: []string{
					"Diseñé y construí una biblioteca de Web Components en TypeScript sin dependencias en tiempo de ejecución que cubre el OWASP Top 10:2021 (A01–A10) con la seguridad activa por defecto: protección CSRF, saneamiento XSS, registro de auditoría y aislamiento con Shadow DOM cerrado, sin que nada de ello requiera configuración por parte del desarrollador",
					"Construí <secure-telemetry-provider>: detección de bots sin CAPTCHA mediante sobres de señales de comportamiento firmados con HMAC-SHA-256 (indicador webdriver, detección de navegador headless, presencia de ratón y teclado, tiempo de envío, dimensiones de pantalla). Las señales se verifican en el servidor y los datos nunca salen de la infraestructura del desarrollador",
					"9 componentes listos para producción publicados en npm: secure-form, secure-input, secure-select, secure-textarea, secure-file-upload, secure-datetime, secure-table, secure-card, secure-telemetry-provider",
					"Independiente de frameworks gracias al estándar W3C Custom Elements: funciona en Angular, React, Vue, Svelte, Go/templ, Django, Rails y HTML puro con una sola etiqueta script",
					"Publiqué el propio sitio de demostración sobre la biblioteca estándar de Go, SQLite, templ y HTML, CSS y TypeScript nativos, desplegado en Fly.io con internacionalización completa (EN/ES/FR/DE)",
				},
			},
			Stack: []string{"Go", "TypeScript", "Web Components", "SQLite", "Fly.io"},
		},
		{
			Name:  "SAUI — Server-Authoritative UI",
			Role:  T{EN: "Own project", ES: "Proyecto propio"},
			Links: []Link{{Label: "saui.fly.dev", URL: "https://saui.fly.dev"}},
			Summary: T{
				EN: "A web architecture where the server owns all state and the browser is a stateless view of it.",
				ES: "Una arquitectura web en la que el servidor posee todo el estado y el navegador es una vista sin estado de él.",
			},
			Bullets: TS{
				EN: []string{
					"Wrote and published the architecture: the gateway contract, the two-layer state model, and the action pattern. The argument is that a UI displaying server state is correct by definition, while one holding its own copy is eventually wrong",
					"Built the reference implementation in Go, SQLite and htmx: no client-side state management, no build step, no framework",
					"Eight-part site (why, architecture, stack, cases, testing, limits, code, blog) shipped in English and Spanish, deployed on Fly.io",
					"Documents where the approach is the wrong choice as plainly as where it fits",
				},
				ES: []string{
					"Escribí y publiqué la arquitectura: el contrato de la pasarela, el modelo de estado en dos capas y el patrón de acciones. El argumento es que una interfaz que muestra el estado del servidor es correcta por definición, mientras que una que guarda su propia copia acaba estando equivocada",
					"Construí la implementación de referencia en Go, SQLite y htmx: sin gestión de estado en el cliente, sin paso de compilación y sin framework",
					"Sitio de ocho secciones (por qué, arquitectura, stack, casos, pruebas, límites, código y blog) publicado en inglés y español, desplegado en Fly.io",
					"Documenta con la misma claridad dónde el enfoque es la elección equivocada y dónde encaja",
				},
			},
			Stack: []string{"Go", "SQLite", "htmx", "Fly.io"},
		},
		{
			Name: "Archway Orthotics Portal",
			Role: T{EN: "Client engagement", ES: "Proyecto para cliente"},
			From: Date{Year: 2026, Month: 6},
			To:   Date{Year: 2026, Month: 7},
			Links: []Link{
				{Label: "archway-orthotics-portal.fly.dev", URL: "https://archway-orthotics-portal.fly.dev"},
			},
			Summary: T{
				EN: "Practitioner-to-lab ordering portal for an Irish medical-device manufacturer, replacing an email and Dropbox workflow.",
				ES: "Portal de pedidos entre profesionales y laboratorio para un fabricante irlandés de productos sanitarios, en sustitución de un flujo de trabajo basado en correo electrónico y Dropbox.",
			},
			Bullets: TS{
				EN: []string{
					"Replaced an email/Dropbox process with a structured portal: practitioners register patients and submit a digital prescription with a 3D foot scan, and the lab moves orders through a manufacturing pipeline and pulls scans for CAD/CAM milling",
					"Handled patient data to GDPR: AES-256-GCM encryption of scan files at rest, per-tenant isolation, CSRF protection, audit logging, fail-closed configuration, and lifecycle handling for storage limitation (Art. 5(1)(e)), erasure (Art. 17) and subject access and portability (Art. 15/20)",
					"Wrote the compliance pack alongside the code: DPIA, records of processing, retention schedule, breach procedure and privacy notice",
					"Strict CSP with no inline scripts or styles; semantic HTML with vanilla JS only for progressive enhancement, plus a self-hosted Three.js viewer for admin scan preview",
					"Around 9,700 lines of Go across 62 files and 13 templ components over 496 commits, on SQLite through a pure-Go driver (no CGO) and deployed to Fly.io",
				},
				ES: []string{
					"Sustituí un proceso basado en correo electrónico y Dropbox por un portal estructurado: los profesionales registran pacientes y envían una prescripción digital con un escaneo 3D del pie, y el laboratorio hace avanzar los pedidos por una cadena de fabricación y descarga los escaneos para el fresado CAD/CAM",
					"Traté datos de pacientes conforme al RGPD: cifrado AES-256-GCM de los archivos de escaneo en reposo, aislamiento por inquilino, protección CSRF, registro de auditoría, configuración a prueba de fallos y gestión del ciclo de vida para la limitación del plazo de conservación (art. 5.1.e), la supresión (art. 17) y el acceso y la portabilidad (arts. 15 y 20)",
					"Redacté la documentación de cumplimiento junto con el código: evaluación de impacto (EIPD), registro de actividades de tratamiento, política de conservación, procedimiento ante brechas y aviso de privacidad",
					"CSP estricta sin scripts ni estilos en línea; HTML semántico con JavaScript nativo solo para mejora progresiva, además de un visor Three.js autoalojado para la previsualización de escaneos en administración",
					"Unas 9.700 líneas de Go en 62 archivos y 13 componentes templ a lo largo de 496 commits, sobre SQLite mediante un controlador puro de Go (sin CGO) y desplegado en Fly.io",
				},
			},
			Stack: []string{"Go", "templ", "SQLite", "AES-256-GCM", "Three.js", "Fly.io"},
		},
		{
			Name: "Archway Orthotics site rebuild",
			Role: T{EN: "Product demo", ES: "Demostración de producto"},
			Summary: T{
				EN: "The flagship demo for SiteForge: a Wix marketing site rebuilt as a fast, self-editable Go site.",
				ES: "La demostración principal de SiteForge: un sitio de marketing en Wix reconstruido como un sitio en Go rápido y editable por el propio cliente.",
			},
			Bullets: TS{
				EN: []string{
					"Rebuilt the public site as server-rendered Go with an HTMX inline editing portal, so the owner can change any word or image in place without touching code",
					"Structured for two audiences at once: clinicians specifying custom orthotics, and patients looking for a refurbishment quote, with the paths diverging early",
					"WCAG 2.2 AA throughout, including a keyboard alternative to drag-reorder and prefers-reduced-motion honoured on every HTMX swap",
				},
				ES: []string{
					"Reconstruí el sitio público en Go renderizado en el servidor con un portal de edición en línea con HTMX, de modo que el propietario puede cambiar cualquier palabra o imagen sin tocar código",
					"Estructurado para dos públicos a la vez: clínicos que prescriben ortesis a medida y pacientes que buscan un presupuesto de renovación, con recorridos que se separan desde el principio",
					"WCAG 2.2 AA en todo el sitio, incluida una alternativa de teclado al reordenamiento por arrastre y el respeto de prefers-reduced-motion en cada intercambio HTMX",
				},
			},
			Stack: []string{"Go", "templ", "SQLite", "HTMX"},
		},
	},
	Skills: []SkillGroup{
		{
			Category: T{EN: "Frontend", ES: "Frontend"},
			Skills: TS{
				EN: []string{"Angular (v1–20)", "TypeScript", "RxJS", "Web Components", "HTML5 / CSS3", "Grid & Flexbox architecture"},
				ES: []string{"Angular (v1–20)", "TypeScript", "RxJS", "Web Components", "HTML5 / CSS3", "Arquitectura con Grid y Flexbox"},
			},
		},
		{
			Category: T{EN: "Backend & Systems", ES: "Backend y sistemas"},
			Skills: TS{
				EN: []string{"Go", "SQLite", "REST APIs", "Fly.io"},
				ES: []string{"Go", "SQLite", "APIs REST", "Fly.io"},
			},
		},
		{
			Category: T{EN: "Security", ES: "Seguridad"},
			Skills: TS{
				EN: []string{"OWASP Top 10", "CSRF / XSS mitigation", "HMAC-SHA-256 signing", "Penetration testing", "Kali Linux", "Burp Suite", "OWASP ZAP", "SQLMap", "Metasploit", "Wireshark"},
				ES: []string{"OWASP Top 10", "Mitigación de CSRF y XSS", "Firma HMAC-SHA-256", "Test de intrusión", "Kali Linux", "Burp Suite", "OWASP ZAP", "SQLMap", "Metasploit", "Wireshark"},
			},
		},
		{
			Category: T{EN: "Practice", ES: "Práctica"},
			Skills: TS{
				EN: []string{"i18n / l10n", "Git", "Component library design", "Framework-agnostic architecture"},
				ES: []string{"i18n / l10n", "Git", "Diseño de bibliotecas de componentes", "Arquitectura independiente de frameworks"},
			},
		},
	},
	Languages: []Language{
		{
			Name:  T{EN: "English", ES: "Inglés"},
			Level: T{EN: "Native", ES: "Nativo"},
		},
		{
			Name: T{EN: "Spanish", ES: "Español"},
			Level: T{
				EN: "Professional working proficiency, 20 years in Madrid",
				ES: "Competencia profesional, 20 años en Madrid",
			},
		},
	},
	Education: []EducationEntry{
		{
			Institution: "cadel.es",
			Program: T{
				EN: "Advanced Cybersecurity & Cyber Intelligence Diploma",
				ES: "Diploma en Ciberseguridad Avanzada e Inteligencia Cibernética",
			},
			From: Date{Year: 2025, Month: 10},
			To:   Date{Year: 2026, Month: 3},
			Detail: TS{
				EN: []string{
					"SOC operations, network defence and attack, GRC, secure development, cyber intelligence",
					"Final project: group penetration test against a deliberately vulnerable web application, documented in English and Spanish",
				},
				ES: []string{
					"Operaciones SOC, defensa y ataque de redes, GRC, desarrollo seguro e inteligencia cibernética",
					"Proyecto final: test de intrusión en grupo contra una aplicación web deliberadamente vulnerable, documentado en inglés y español",
				},
			},
		},
		{
			Institution: "Ondas Formación",
			Program: T{
				EN: "Web Application Development — HTML5, CSS3, JavaScript, jQuery",
				ES: "Desarrollo de Aplicaciones Web — HTML5, CSS3, JavaScript, jQuery",
			},
			From: Date{Year: 2012, Month: 9},
			To:   Date{Year: 2012, Month: 12},
		},
	},
}
