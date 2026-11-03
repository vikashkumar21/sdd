package check

import (
	"github.com/vikashkumar21/sdd/internal/skill"
)

func (c *checker) checkAgentSkills() {
	skills, err := skill.Discover(c.root)
	if err != nil {
		c.add(Error, "skills", 0, "skill-discovery", "failed to scan skills directory: %v", err)
		return
	}

	for _, sk := range skills {
		for _, vErr := range sk.Errors {
			c.add(Error, sk.File, vErr.Line, "skill-"+vErr.Field, "%s", vErr.Message)
		}
	}
}
