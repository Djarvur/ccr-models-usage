package provider

// Registry is the lookup table of known Adapters, indexed by the host
// they service. A single registry instance is created in main and
// passed to the parallel fetcher.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]Adapter)}
}

// Register adds an adapter to the registry, keyed on the adapter's
// Host(). Registering two adapters for the same host overwrites the
// earlier one; this is a programmer error but the latest call wins.
func (r *Registry) Register(adapter Adapter) {
	r.adapters[adapter.Host()] = adapter
}

// Match returns the adapter registered for host, or nil if no adapter
// matches.
func (r *Registry) Match(host string) Adapter {
	return r.adapters[host]
}

// Hosts returns the set of hosts registered. The order is not
// specified.
func (r *Registry) Hosts() []string {
	out := make([]string, 0, len(r.adapters))
	for host := range r.adapters {
		out = append(out, host)
	}

	return out
}
