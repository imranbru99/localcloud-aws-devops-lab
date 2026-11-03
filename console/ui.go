package console

import _ "embed"

//go:embed public/index.html
var IndexHTML string

//go:embed public/static/tailwind.min.js
var TailwindJS string

//go:embed public/academy.html
var AcademyHTML string

//go:embed public/curriculum.json
var CurriculumJSON []byte
