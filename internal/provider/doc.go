// Package provider defines the provider-agnostic interface and types for
// discovering image models and generating images. Callers depend only on this
// package, never on provider-specific request or response formats. Capabilities
// describe reference requirements and output formats; shared header validation
// keeps UI feedback and provider-boundary checks consistent.
package provider
