package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/client9/gospell"
)

func addOrPanic(dict map[string]string, key, value string) {
	if _, ok := dict[key]; ok {
		log.Printf("Already have %q", key)
	}

	// this happens for captialization rules
	//
	// english->English
	//
	// variations will generate
	//
	// engish->English, English->English, ENGLISH->ENGLISH
	//
	// so we ignore them
	if key == value {
		return
	}

	dict[key] = value
}

func mergeDict(a, b map[string]string) {
	for k, v := range b {
		addOrPanic(a, k, v)
	}
}

func removeCase(inmap map[string]string, word string) {
	style := gospell.CaseStyle(word)
	kcase := gospell.CaseVariations(word, style)
	for i := 0; i < len(kcase); i++ {
		delete(inmap, kcase[i])
	}
}
func expandCase(inmap map[string]string) map[string]string {
	out := make(map[string]string, len(inmap)*3)
	for k, v := range inmap {
		style := gospell.CaseStyle(k)
		kcase := gospell.CaseVariations(k, style)
		vcase := gospell.CaseVariations(v, style)
		for i := 0; i < len(kcase); i++ {
			addOrPanic(out, kcase[i], vcase[i])
		}
	}
	return out
}

func parseWikipediaFormat(text string) map[string]string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	dict := make(map[string]string, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.Split(line, "->")
		if len(parts) != 2 {
			log.Fatalf(fmt.Sprintf("failed parsing %q", line))
		}
		spellings := strings.Split(parts[1], ",")
		dict[parts[0]] = strings.TrimSpace(spellings[0])
	}
	return dict
}

func main() {
	fmt.Printf("package misspell\n\n")

	// create main word list
	dict := make(map[string]string)
	mergeDict(dict, dictWikipedia())
	mergeDict(dict, dictAdditions())
	words := make([]string, 0, len(dict))
	for k := range dict {
		words = append(words, k)
	}
	sort.Strings(words)
	fmt.Printf("// DictMain is the main rule set, not including locale-specific spellings\n")
	fmt.Printf("var DictMain = []string{\n")
	for _, word := range words {
		fmt.Printf("\t%q, %q,\n", word, dict[word])
	}
	fmt.Printf("}\n\n")

	dict = make(map[string]string)
	mergeDict(dict, dictAmerican())
	words = make([]string, 0, len(dict))
	for k := range dict {
		words = append(words, k)
	}
	sort.Strings(words)
	fmt.Printf("// DictAmerican converts UK spellings to US spellings\n")
	fmt.Printf("var DictAmerican = []string{\n")
	for _, word := range words {
		fmt.Printf("\t%q, %q,\n", word, dict[word])
	}
	fmt.Printf("}\n\n")

	dict = make(map[string]string)
	mergeDict(dict, dictBritish())
	words = make([]string, 0, len(dict))
	for k := range dict {
		words = append(words, k)
	}
	sort.Strings(words)
	fmt.Printf("// DictBritish converts US spellings to UK spellings\n")
	fmt.Printf("var DictBritish = []string{\n")
	for _, word := range words {
		fmt.Printf("\t%q, %q,\n", word, dict[word])
	}
	fmt.Printf("}\n")
}
