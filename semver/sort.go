package semver

import "sort"

// Versions represents a list of versions to be sorted.
type Versions []Version

func (v Versions) Len() int {
	return len(v)
}

func (v Versions) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v Versions) Less(i, j int) bool {
	return v[i].Compare(v[j]) == -1
}

func Sort(versions []Version) {
	sort.Sort(Versions(versions))
}
