package main

//////////////////////
// LOL YES GENERICS //
//////////////////////

var member struct{}

type Map[K comparable, V any] map[K]V

type theme_info Map[string, string]

type set Map[string, struct{}]

// generic methods, kinda

func (t theme_info) Map() Map[string, string] {
	return Map[string, string](t)
}

func (s set) Map() Map[string, struct{}] {
	return Map[string, struct{}](s)
}

func (m Map[K, V]) contains_key(key K) bool {
	for k := range m {
		if k == key {
			return true
		}
	}
	return false
}

func (m Map[K, V]) contains_at_least_one_key(keys set) bool {
	for key := range m {
		if m.contains_key(key) {
			return true
		}
	}
	return false
}

func (m Map[K, V]) contains_all_keys(keys []K) (bool, []K) {
	var not_contained []K

	for _, key := range keys {
		if !m.contains_key(key) {
			not_contained = append(not_contained, key)
		}
	}
	if len(not_contained) > 0 {
		return false, not_contained
	}

	return true, nil
}
