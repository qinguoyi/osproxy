package utils

import "reflect"

// RemoveDuplicates .
func RemoveDuplicates(arr []string) []string {
	encountered := map[string]bool{}
	var result []string

	for _, v := range arr {
		if encountered[v] == true {
			continue
		} else {
			encountered[v] = true
			result = append(result, v)
		}
	}

	return result
}

// Contains .
func Contains(search interface{}, target interface{}) bool {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == search {
				return true
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(search)).IsValid() {
			return true
		}
	}
	return false
}
