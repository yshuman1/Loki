package main

import (
	"fmt"
	"reflect"

	"github.com/emersion/go-imap/v2"
)

func main() {
	t := reflect.TypeOf(imap.Address{})
	fmt.Printf("Address fields:\n")
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fmt.Printf("- %s: %s\n", field.Name, field.Type)
	}
}
