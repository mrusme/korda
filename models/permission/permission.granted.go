package permission

import "slices"

func (Permission) Enum() []string {
	return values
}

func Values() []string {
	return slices.Clone(values)
}

func IsGranted(requested []Permission, current []Permission) bool {
	if slices.Contains(current, All) {
		return true
	}

	for _, p := range requested {
		if !slices.Contains(current, p) {
			return false
		}
	}

	return true
}
