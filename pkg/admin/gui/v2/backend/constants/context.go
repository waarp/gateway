package constants

type ContextKey string

const (
	ContextLanguageKey ContextKey = "language"
	ContextUserKey     ContextKey = "user"
	// ContextEnvironmentKey holds the environment label of the instance, which
	// the authentication joins to the session.
	ContextEnvironmentKey ContextKey = "environment"
)
