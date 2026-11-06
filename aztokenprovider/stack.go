package aztokenprovider

import (
	"context"
	"regexp"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// stackNamespacePattern matches the namespace Grafana Cloud reports for a stack.
var stackNamespacePattern = regexp.MustCompile(`^stacks-([0-9]+)$`)

// stackIDFromContext returns the Grafana Cloud stack ID that Grafana reports for the request in
// the plugin context's namespace ("stacks-<id>"), or "" when the namespace is not a stack's.
// Single-tenant and local Grafana report "default" or "org-<n>", and Grafana before 13 reports
// nothing, so the stack ID is only ever taken from a namespace that is exactly a stack's.
//
// Grafana derives the namespace from the authenticated tenant, so unlike a configured stack ID it
// cannot be mistyped and it is correct per request in a process that serves several tenants.
func stackIDFromContext(ctx context.Context) string {
	m := stackNamespacePattern.FindStringSubmatch(backend.PluginConfigFromContext(ctx).Namespace)
	if m == nil {
		return ""
	}
	return m[1]
}
