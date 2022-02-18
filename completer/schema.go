package completer

import (
	"sort"
)

func stringsUniq(all []string) (result []string) {

	indexMap := make(map[string]int)

	for i, item := range all {
		indexMap[item] = i
	}

	for str := range indexMap {
		result = append(result, str)
	}

	sort.Strings(result)

	return
}
