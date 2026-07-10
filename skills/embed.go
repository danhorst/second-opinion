// Package skills embeds the canonical calling-agent skill so the installed
// binary can produce it without repository access. The repository file is
// the source of truth by construction.
package skills

import _ "embed"

//go:embed second-opinion/SKILL.md
var SecondOpinion string
