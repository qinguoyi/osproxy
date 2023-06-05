package base

func Query(values map[string][]string, name string) string {
	res := ""
	resList, ok := values[name]
	if ok {
		res = resList[0]
	}
	return res
}
