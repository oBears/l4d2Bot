package utils

func GetStrArg(matches []string, pos int) string {
	if len(matches) > pos {
		return matches[pos]
	}
	return ""
}
