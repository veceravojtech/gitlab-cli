package mcp

import "errors"

// Config errors
var (
	ErrConfigLoad     = errors.New("configuration load failed")
	ErrConfigValidate = errors.New("configuration validation failed")
	ErrAuthMissing    = errors.New("authentication credentials missing")
)

// GitLab API errors
var (
	ErrGitLabAPI       = errors.New("gitlab API error")
	ErrMRNotFound      = errors.New("merge request not found")
	ErrProjectNotFound = errors.New("project not found")
)

// Validation errors
var (
	ErrInvalidInput = errors.New("invalid input")
	ErrMissingParam = errors.New("missing required parameter")
)

// Merge operation errors
var (
	ErrMergeConflict  = errors.New("merge conflict")
	ErrMergeTimeout   = errors.New("merge timeout exceeded")
	ErrRebaseFailed   = errors.New("rebase failed")
	ErrPipelineFailed = errors.New("pipeline failed")
)
