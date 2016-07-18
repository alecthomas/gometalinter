package main

func addLeftDelimiter(dict map[string]string, word string) {
	if val, ok := dict[word]; ok {
		delete(dict, word)
		dict[" "+word] = " " + val
	}
}
func addRightDelimiter(dict map[string]string, word string) {
	if val, ok := dict[word]; ok {
		delete(dict, word)
		dict[word+" "] = val + " "
	}
}
func addBothDelimiter(dict map[string]string, word string) {
	if val, ok := dict[word]; ok {
		delete(dict, word)
		dict[" "+word+" "] = " " + val + " "
	}
}
