package provider_errors

import "errors"

var (
	ErrProviderUnauthorized    = errors.New("provider unauthorized")
	ErrProviderRateLimited     = errors.New("provider rate limited")
	ErrProviderTimedOut        = errors.New("provider timed out")
	ErrProviderOperationFailed = errors.New("provider operation failed")
	ErrProviderMisconfigured   = errors.New("provider misconfigured")
	ErrAssetDownloadFailed     = errors.New("asset download failed")
)
