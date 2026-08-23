// Package data holds Barry Prendergast's CV content as plain Go values.
// It is the single source of truth the site templates render from.
package data

// Contact holds how to reach Barry.
type Contact struct {
	Email    string
	Phone    string
	LinkedIn string
	GitHub   string
	Site     string
	Location string
}

// Link is a labelled destination. Jobs can carry several: the Secure UI
// Components work lives at a showcase site, a repository and an npm package,
// and a single URL field could only point at one of them.
type Link struct {
	Label string
	URL   string
}

// Language is a spoken language and the level claimed for it.
type Language struct {
	Name  string
	Level string
}

// Job is one entry in the experience timeline.
type Job struct {
	Company  string
	Title    string
	Start    string
	End      string // "Present" for the current role
	Location string
	Summary  string // optional one-line context; empty if not needed
	Bullets  []string
	Stack    []string
	Links    []Link // where the work can actually be seen, if anywhere

	// Kind separates an employer relationship from the entries that are not
	// one. Empty means employment, so every real role is unchanged by this
	// field existing. Anything else must never be rendered, or described in
	// structured data, as a job.
	Kind string // "", "independent", "break"
}

// IsEmployment reports whether this entry represents working for someone else.
func (j Job) IsEmployment() bool { return j.Kind == "" }

// Project is one piece of work on the projects page. Unlike a Job it may have
// no employer and no dates — what matters is what it is and where to see it.
type Project struct {
	Name    string
	Role    string // "Client engagement", "Own project", "Product demo"
	Period  string // optional; empty for ongoing or undated work
	Links   []Link // public destinations, empty when there is nothing to show
	Summary string
	Bullets []string
	Stack   []string
}

// SkillGroup is a named cluster of skills shown together.
type SkillGroup struct {
	Category string
	Skills   []string
}

// EducationEntry is one line of education or certification.
type EducationEntry struct {
	Institution string
	Program     string
	Start       string
	End         string
	Detail      []string // syllabus, final project, anything worth naming
}

// Me is the single source of truth for the site's content.
var Me = struct {
	Name     string
	Headline string
	Tagline  string
	// Status is the line on the home page saying what is in flight. It lives
	// here rather than in the template because it has to stay consistent with
	// Jobs and Projects, and copy kept in a template drifts out of step with
	// the data it describes.
	Status    string
	Contact   Contact
	Jobs      []Job
	Projects  []Project
	Skills    []SkillGroup
	Education []EducationEntry
	Languages []Language
}{
	Name:     "Barry Prendergast",
	Headline: "Senior Full-Stack Engineer",
	Tagline:  "Fifteen years of Angular at enterprise scale. Now building security-first platforms in Go.",
	Status:   "building saui and secure-ui-components · shipped the archway orthotics portal",
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
		Location: "Madrid, Spain",
	},
	Jobs: []Job{
		{
			Company:  "Independent",
			Title:    "Open source, client engagements and cybersecurity training",
			Kind:     "independent",
			Start:    "March 2026",
			End:      "Present",
			Location: "Madrid",
			Bullets: []string{
				"Built and published Secure UI Components and SAUI, and delivered the Archway Orthotics practitioner portal as a client engagement (see Projects)",
				"Completed the Advanced Cybersecurity & Cyber Intelligence diploma at CADEL, covering penetration testing methodology, SOC analysis, and the Kali Linux, Burp Suite, OWASP ZAP, SQLMap and Metasploit toolchain",
			},
			Stack: []string{"Go", "TypeScript", "Web Components", "SQLite", "Fly.io"},
		},
		{
			Company:  "Quality Compusoft",
			Title:    "Senior Frontend Developer",
			Start:    "September 2020",
			End:      "March 2026",
			Location: "Madrid",
			Summary:  "Principal frontend architect for Telefy, an enterprise media management platform for dental and veterinary TV networks.",
			Bullets: []string{
				"Architected a complete Angular application from the ground up (v13 → v18 → v20): a dual-interface platform with an admin dashboard and a user-facing app covering device management, real-time streaming, drag-and-drop playlist creation, and social media integrations",
				"Created 115+ custom components, including a 27-component UI library built without external frameworks (inputs, selects, dropzones, modals, sliders, media viewers), with hand-coded responsive CSS on Grid/Flexbox, cutting bundle size 40%",
				"Implemented the frontend UI and user flows for social integrations (Facebook, Instagram, TikTok, YouTube, X/Twitter): OAuth authorization screens, connection status, content feed views, and posting workflows, in collaboration with the backend team",
				"Shipped i18n across 4 languages (Spanish, English, Catalan, Portuguese) with dynamic switching, a real-time streaming interface, and remote device configuration across multiple TV devices",
				"Led a refactor removing layout wrappers from 50+ components and upgraded the legacy app from Angular v12 to v17 (dependency-limited) while evolving Telefy itself through v13 to v20",
			},
			Stack: []string{"Angular 13-20", "TypeScript", "RxJS", "CSS Grid/Flexbox", "REST APIs"},
		},
		{
			Company:  "Career break",
			Title:    "Parental",
			Kind:     "break",
			Start:    "July 2019",
			End:      "September 2020",
			Location: "Madrid",
			Summary:  "Full-time carer for two young children.",
		},
		{
			Company:  "Arcmedia AG",
			Title:    "Senior Front-End Developer",
			Start:    "July 2017",
			End:      "July 2019",
			Location: "Madrid",
			Bullets: []string{
				"Delivered responsive layouts across client projects, mainly in Drupal, with one project in Angular 8",
				"Configured and implemented email templates with Foundation for Emails",
			},
			Stack: []string{"Drupal", "Angular 8", "Foundation"},
		},
		{
			Company:  "GRADDO",
			Title:    "Senior Frontend Developer",
			Start:    "February 2016",
			End:      "June 2017",
			Location: "Madrid",
			Bullets: []string{
				"Built the UI layer for in-house .NET applications with Bootstrap",
				"Worked on a DevOps team building a banking app in Angular 1.5.8",
			},
			Stack: []string{".NET", "Bootstrap", "Angular 1.5"},
		},
		{
			Company:  "Cecabank",
			Title:    "Front-End Developer",
			Start:    "April 2015",
			End:      "February 2016",
			Location: "Madrid",
			Bullets: []string{
				"Built and maintained layouts for banking-sector projects in HTML, CSS, and JavaScript",
			},
			Stack: []string{"HTML", "CSS", "JavaScript", "Bootstrap", ".NET"},
		},
		{
			Company:  "Vivocom",
			Title:    "Front-End Developer",
			Start:    "September 2014",
			End:      "February 2015",
			Location: "Madrid",
			Bullets: []string{
				"Maquetación web for Vivocom Keepunto: converted PSD designs into responsive HTML, CSS/LESS, Bootstrap and jQuery for banking-sector apps and promotional sites",
			},
			Stack: []string{"HTML", "CSS", "LESS", "Bootstrap", "jQuery"},
		},
		{
			Company:  "Vocento",
			Title:    "Front-End Developer",
			Start:    "April 2013",
			End:      "September 2014",
			Location: "Madrid",
			Bullets: []string{
				"Built responsive, mobile-first web layouts with HTML5, CSS3, and jQuery for digital media properties at one of Spain's largest media groups, including GUAPABOX (fashion e-commerce) and ABC FOTO (photo platform)",
				"Produced responsive templates with Skeleton Grid and Bootstrap, plus landing-page templates for several products",
			},
			Stack: []string{"HTML5", "CSS3", "jQuery", "Bootstrap"},
		},
		{
			Company:  "Mercantis",
			Title:    "Web Layout Developer",
			Start:    "2008",
			End:      "2011",
			Location: "",
			Bullets: []string{
				"Built web layouts and front-end for health-sector products, and maintained the WordPress sites they ran on",
			},
			Stack: []string{"HTML", "CSS", "jQuery", "WordPress"},
		},
	},
	Projects: []Project{
		{
			Name:    "Secure UI Components",
			Role:    "Own project",
			Summary: "Zero-dependency, security-first Web Components library and its Go-built showcase site.",
			Links: []Link{
				{Label: "secure-ui-web.fly.dev", URL: "https://secure-ui-web.fly.dev"},
				{Label: "github.com/Barryprender/Secure-UI", URL: "https://github.com/Barryprender/Secure-UI"},
				{Label: "npm: secure-ui-components", URL: "https://www.npmjs.com/package/secure-ui-components"},
			},
			Bullets: []string{
				"Designed and built a zero-runtime-dependency TypeScript Web Components library covering OWASP Top 10:2021 (A01–A10) with security active by default: CSRF protection, XSS sanitisation, audit logging, and closed Shadow DOM isolation, none of it requiring developer configuration",
				"Built <secure-telemetry-provider>: CAPTCHA-free bot detection using HMAC-SHA-256 signed behavioural signal envelopes (webdriver flag, headless detection, mouse/keyboard presence, submit timing, screen dimensions). Signals are verified server-side, and the data never leaves the developer's infrastructure",
				"9 production-ready components published to npm: secure-form, secure-input, secure-select, secure-textarea, secure-file-upload, secure-datetime, secure-table, secure-card, secure-telemetry-provider",
				"Framework-agnostic via the W3C Custom Elements standard: works in Angular, React, Vue, Svelte, Go/templ, Django, Rails, and plain HTML with a single script tag",
				"Shipped the showcase site itself on the Go standard library, SQLite, templ, and native HTML/CSS/TypeScript, deployed to Fly.io with full i18n (EN/ES/FR/DE)",
			},
			Stack: []string{"Go", "TypeScript", "Web Components", "SQLite", "Fly.io"},
		},
		{
			Name:    "SAUI — Server-Authoritative UI",
			Role:    "Own project",
			Links:   []Link{{Label: "saui.fly.dev", URL: "https://saui.fly.dev"}},
			Summary: "A web architecture where the server owns all state and the browser is a stateless view of it.",
			Bullets: []string{
				"Wrote and published the architecture: the gateway contract, the two-layer state model, and the action pattern. The argument is that a UI displaying server state is correct by definition, while one holding its own copy is eventually wrong",
				"Built the reference implementation in Go, SQLite and htmx: no client-side state management, no build step, no framework",
				"Eight-part site (why, architecture, stack, cases, testing, limits, code, blog) shipped in English and Spanish, deployed on Fly.io",
				"Documents where the approach is the wrong choice as plainly as where it fits",
			},
			Stack: []string{"Go", "SQLite", "htmx", "Fly.io"},
		},
		{
			Name:    "Archway Orthotics Portal",
			Role:    "Client engagement",
			Period:  "June 2026 — July 2026",
			Summary: "Practitioner-to-lab ordering portal for an Irish medical-device manufacturer, replacing an email and Dropbox workflow.",
			Bullets: []string{
				"Replaced an email/Dropbox process with a structured portal: practitioners register patients and submit a digital prescription with a 3D foot scan, and the lab moves orders through a manufacturing pipeline and pulls scans for CAD/CAM milling",
				"Handled patient data to GDPR: AES-256-GCM encryption of scan files at rest, per-tenant isolation, CSRF protection, audit logging, fail-closed configuration, and lifecycle handling for storage limitation (Art. 5(1)(e)), erasure (Art. 17) and subject access and portability (Art. 15/20)",
				"Wrote the compliance pack alongside the code: DPIA, records of processing, retention schedule, breach procedure and privacy notice",
				"Strict CSP with no inline scripts or styles; semantic HTML with vanilla JS only for progressive enhancement, plus a self-hosted Three.js viewer for admin scan preview",
				"Around 9,700 lines of Go across 62 files and 13 templ components over 496 commits, on SQLite through a pure-Go driver (no CGO) and deployed to Fly.io",
			},
			Stack: []string{"Go", "templ", "SQLite", "AES-256-GCM", "Three.js", "Fly.io"},
		},
		{
			Name:    "Archway Orthotics site rebuild",
			Role:    "Product demo",
			Summary: "The flagship demo for SiteForge: a Wix marketing site rebuilt as a fast, self-editable Go site.",
			Bullets: []string{
				"Rebuilt the public site as server-rendered Go with an HTMX inline editing portal, so the owner can change any word or image in place without touching code",
				"Structured for two audiences at once: clinicians specifying custom orthotics, and patients looking for a refurbishment quote, with the paths diverging early",
				"WCAG 2.2 AA throughout, including a keyboard alternative to drag-reorder and prefers-reduced-motion honoured on every HTMX swap",
			},
			Stack: []string{"Go", "templ", "SQLite", "HTMX"},
		},
	},
	Skills: []SkillGroup{
		{Category: "Frontend", Skills: []string{"Angular (v1–20)", "TypeScript", "RxJS", "Web Components", "HTML5 / CSS3", "Grid & Flexbox architecture"}},
		{Category: "Backend & Systems", Skills: []string{"Go", "SQLite", "REST APIs", "Fly.io"}},
		{Category: "Security", Skills: []string{"OWASP Top 10", "CSRF / XSS mitigation", "HMAC-SHA-256 signing", "Penetration testing", "Kali Linux", "Burp Suite", "OWASP ZAP", "SQLMap", "Metasploit", "Wireshark"}},
		{Category: "Practice", Skills: []string{"i18n / l10n", "Git", "Component library design", "Framework-agnostic architecture"}},
	},
	Languages: []Language{
		{Name: "English", Level: "Native"},
		{Name: "Spanish", Level: "Professional working proficiency, 20 years in Madrid"},
	},
	Education: []EducationEntry{
		{
			Institution: "cadel.es",
			Program:     "Advanced Cybersecurity & Cyber Intelligence Diploma",
			Start:       "October 2025",
			End:         "March 2026",
			Detail: []string{
				"SOC operations, network defence and attack, GRC, secure development, cyber intelligence",
				"Final project: group penetration test against a deliberately vulnerable web application, documented in English and Spanish",
			},
		},
		{Institution: "Ondas Formación", Program: "Web Application Development — HTML5, CSS3, JavaScript, jQuery", Start: "2012", End: "2012"},
	},
}
