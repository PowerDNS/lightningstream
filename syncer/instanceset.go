package syncer

import (
	"sort"
	"strings"
)

func NewInstanceSet() *InstanceSet {
	return &InstanceSet{
		m: map[string]bool{},
	}
}

// InstanceSet is a set of instances that we are still waiting for
type InstanceSet struct {
	m map[string]bool
}

func (s *InstanceSet) Add(name string) {
	s.m[name] = true
}

func (s *InstanceSet) Remove(name string) {
	delete(s.m, name)
}

func (s *InstanceSet) Done() bool {
	return len(s.m) == 0
}

func (s *InstanceSet) Contains(name string) bool {
	return s.m[name]
}

func (s *InstanceSet) List() []string {
	var instances []string
	for instance := range s.m {
		instances = append(instances, instance)
	}
	sort.Strings(instances)
	return instances
}

func (s *InstanceSet) String() string {
	return strings.Join(s.List(), " ")
}

func (s *InstanceSet) CleanDisappeared(seen []string) (cleaned []string) {
	current := make(map[string]bool)
	for _, instance := range seen {
		current[instance] = true
	}
	// No need to wait for these any longer, remove them from the map
	var toDelete []string
	for instance := range s.m {
		if !current[instance] {
			toDelete = append(toDelete, instance)
		}
	}
	for _, instance := range toDelete {
		s.Remove(instance)
	}
	sort.Strings(toDelete)
	return toDelete
}
