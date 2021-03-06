/*
Copyright 2019 The HAProxy Ingress Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"fmt"
	"reflect"
	"sort"
)

// CreateHosts ...
func CreateHosts() *Hosts {
	return &Hosts{
		items:    map[string]*Host{},
		itemsAdd: map[string]*Host{},
		itemsDel: map[string]*Host{},
	}
}

// CreatePathLink ...
func CreatePathLink(hostname, path string) PathLink {
	return PathLink{
		hostname: hostname,
		path:     path,
	}
}

// AcquireHost ...
func (h *Hosts) AcquireHost(hostname string) *Host {
	if host := h.FindHost(hostname); host != nil {
		return host
	}
	host := h.createHost(hostname)
	h.items[hostname] = host
	h.itemsAdd[hostname] = host
	return host
}

// FindHost ...
func (h *Hosts) FindHost(hostname string) *Host {
	return h.items[hostname]
}

// RemoveAll ...
func (h *Hosts) RemoveAll(hostnames []string) {
	for _, hostname := range hostnames {
		if item, found := h.items[hostname]; found {
			h.releaseHost(item)
			h.itemsDel[hostname] = item
			delete(h.items, hostname)
		}
	}
}

// Shrink removes matching added and deleted hosts from the changing hashmap
// tracker that has the same content. A matching added+deleted pair means
// that a hostname was reparsed but its content wasn't changed.
func (h *Hosts) Shrink() {
	for name, del := range h.itemsDel {
		if add, found := h.itemsAdd[name]; found {
			if reflect.DeepEqual(add, del) {
				h.items[name] = del
				delete(h.itemsAdd, name)
				delete(h.itemsDel, name)
			}
		}
	}
}

// Commit ...
func (h *Hosts) Commit() {
	h.itemsAdd = map[string]*Host{}
	h.itemsDel = map[string]*Host{}
	h.hasCommit = true
}

// HasCommit ...
func (h *Hosts) HasCommit() bool {
	return h.hasCommit
}

// Changed ...
func (h *Hosts) Changed() bool {
	return len(h.itemsAdd) > 0 || len(h.itemsDel) > 0
}

func (h *Hosts) createHost(hostname string) *Host {
	return &Host{
		Hostname: hostname,
		hosts:    h,
	}
}

// BuildSortedItems ...
func (h *Hosts) BuildSortedItems() []*Host {
	items := make([]*Host, len(h.items))
	var i int
	for hostname, item := range h.items {
		if hostname != DefaultHost {
			items[i] = item
			i++
		}
	}
	items = items[:i]
	sort.Slice(items, func(i, j int) bool {
		return items[i].Hostname < items[j].Hostname
	})
	if len(items) == 0 {
		return nil
	}
	return items
}

// Items ...
func (h *Hosts) Items() map[string]*Host {
	return h.items
}

// ItemsAdd ...
func (h *Hosts) ItemsAdd() map[string]*Host {
	return h.itemsAdd
}

// ItemsDel ...
func (h *Hosts) ItemsDel() map[string]*Host {
	return h.itemsDel
}

// DefaultHost ...
func (h *Hosts) DefaultHost() *Host {
	return h.items[DefaultHost]
}

// releaseHost does a reverse update on the Hosts state
// due to the removal of a Host item
func (h *Hosts) releaseHost(host *Host) {
	if host.sslPassthrough {
		h.sslPassthroughCount--
	}
}

// HasSSLPassthrough ...
func (h *Hosts) HasSSLPassthrough() bool {
	return h.sslPassthroughCount > 0
}

// HasVarNamespace ...
func (h *Hosts) HasVarNamespace() bool {
	for _, host := range h.items {
		if host.VarNamespace {
			return true
		}
	}
	return false
}

// FindPath ...
func (h *Host) FindPath(path string) *HostPath {
	for _, p := range h.Paths {
		if p.Path == path {
			return p
		}
	}
	return nil
}

// AddPath ...
func (h *Host) AddPath(backend *Backend, path string, match MatchType) {
	link := CreatePathLink(h.Hostname, path)
	var hback HostBackend
	if backend != nil {
		hback = HostBackend{
			ID:        backend.ID,
			Namespace: backend.Namespace,
			Name:      backend.Name,
			Port:      backend.Port,
		}
		backend.AddBackendPath(link)
	} else {
		hback = HostBackend{ID: "_error404"}
	}
	h.Paths = append(h.Paths, &HostPath{
		Path:    path,
		Link:    link,
		Match:   match,
		Backend: hback,
	})
	// reverse order in order to avoid overlap of sub-paths
	sort.Slice(h.Paths, func(i, j int) bool {
		return h.Paths[i].Path > h.Paths[j].Path
	})
}

// HasTLSAuth ...
func (h *Host) HasTLSAuth() bool {
	return h.TLS.CAHash != ""
}

// SSLPassthrough ...
func (h *Host) SSLPassthrough() bool {
	return h.sslPassthrough
}

// SetSSLPassthrough ...
func (h *Host) SetSSLPassthrough(value bool) {
	if h.sslPassthrough == value {
		return
	}
	if value {
		h.hosts.sslPassthroughCount++
	} else {
		h.hosts.sslPassthroughCount--
	}
	h.sslPassthrough = value
}

// IsEmpty ...
func (l *PathLink) IsEmpty() bool {
	return l.hostname == "" && l.path == ""
}

// Less ...
func (l *PathLink) Less(other PathLink, reversePath bool) bool {
	if l.hostname == other.hostname {
		if reversePath {
			return l.path > other.path
		}
		return l.path < other.path
	}
	return l.hostname < other.hostname
}

// String ...
func (f *Frontend) String() string {
	return fmt.Sprintf("%+v", *f)
}

// String ...
func (h *Host) String() string {
	return fmt.Sprintf("%+v", *h)
}

// HasTLS ...
func (h *HostTLSConfig) HasTLS() bool {
	return h.TLSFilename != ""
}
