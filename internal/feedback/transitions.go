package feedback

import (
	"mindflow/internal/release"
	"regexp"
	"strings"
)

// ReviewCommand selects an allowed transition and records its explanation in the discussion.
type ReviewCommand struct {
	Action  string `json:"action"`
	Comment string `json:"comment"`
}

// Decision contains the next state and its discussion entry.
type Decision struct{ Status, Comment, Release string }

// Decide enforces the report lifecycle without performing storage operations.
func Decide(state string, command ReviewCommand) (Decision, error) {
	result := Decision{Comment: strings.TrimSpace(command.Comment)}
	switch {
	case state == StateCreated && command.Action == ActionAccept:
		result.Status = StateAccepted
		if result.Comment == "" {
			result.Comment = "Accepted, to do."
		}
	case state == StateCreated && command.Action == ActionDeny:
		result.Status = StateClosed
	case state == StateAccepted && (command.Action == ActionCloseRelease || command.Action == ActionCloseComment):
		result.Status = StateClosed
		if command.Action == ActionCloseRelease {
			for _, version := range versionInComment.FindAllString(result.Comment, -1) {
				if release.Valid(version) {
					result.Release = version

					break
				}
			}
			if result.Release == "" {
				return Decision{}, invalid("Include the release version in your comment, for example: Fixed in v1.2.0.")
			}
		}
	case state == StateClosed && command.Action == ActionReopen:
		result.Status = StateCreated
		if result.Comment == "" {
			result.Comment = "Reopened."
		}
	default:
		return Decision{}, invalid("This action is not available in the report's current state.")
	}
	comment, err := text(result.Comment, commentLimit)
	if err != nil {
		return Decision{}, invalid("Add a comment explaining this decision (up to 5,000 characters).")
	}
	result.Comment = comment

	return result, nil
}

var versionInComment = regexp.MustCompile(`\bv[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?\b`)

func availableActions(state string) []string {
	switch state {
	case StateCreated:
		return []string{ActionAccept, ActionDeny}
	case StateAccepted:
		return []string{ActionCloseRelease, ActionCloseComment}
	case StateClosed:
		return []string{ActionReopen}
	default:
		return []string{}
	}
}

// Report states and supported administrative transition actions.
const (
	StateCreated       = "created"
	StateAccepted      = "accepted"
	StateClosed        = "closed"
	ActionAccept       = "accept"
	ActionDeny         = "deny"
	ActionCloseRelease = "close_release"
	ActionCloseComment = "close_comment"
	ActionReopen       = "reopen"
)
