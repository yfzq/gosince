package main

import (
	"testing"
)

func TestValidName(t *testing.T) {
	input := map[string]bool{
		"abc":  true,
		"a1":   true,
		"Open": true,
		"ab&":  false,
		"ab-":  false,
		"_ab":  true}
	for k, v := range input {
		err := isValidName(k)
		if v == true && err != nil {
			t.Errorf("%v", err)
		} else if v == false && err == nil {
			t.Errorf("%v should be invalid", k)
		}
	}
}
