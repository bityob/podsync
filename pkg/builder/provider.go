package builder

import "github.com/mxpv/podsync/pkg/model"

// RequiresToken reports whether the given provider needs an API token to operate.
func RequiresToken(provider model.Provider) bool {
	switch provider {
	case model.ProviderYoutube, model.ProviderVimeo, model.ProviderTwitch:
		return true
	default:
		return false
	}
}
