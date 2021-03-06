// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package lookslike

import (
	"fmt"
	"strings"

	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
)

func Example() {
	// Let's say we want to validate this map
	data := map[string]interface{}{"foo": "bar", "baz": "bot", "count": 1}

	// We can validate the data by creating a lookslike.Validator
	// validator.Validators are functions created by compiling the special lookslike.map[string]interface{}
	// type. This is a map[string]interface{} that can be compiled
	// into a series of checks.
	//

	// We can validate the data by defining a validator for this data.
	//Lookslike has powerful matching features for maps and slices especially.
	// You can see an example validator below:
	validator := MustCompile(map[string]interface{}{
		"foo": isdef.IsStringContaining("a"),
		"baz": "bot",
	})

	// When being used in test-suites, you should use testslike.Test to execute the validator
	// This produces easy to read test output, and outputs one failed assertion per failed matcher
	// See the docs for testslike for more info.
	// testslike.Test(t, validator, data)

	// If you need more control than testslike.Test provides, you can use the results directly
	results := validator(data)

	// The Results.Valid property indicates if the validator passed
	fmt.Printf("Results.Valid: %t\n", results.Valid)

	// Results.Errors() returns one error per failed match
	fmt.Printf("There were %d errors\n", len(results.Errors()))

	// Results.Fields is a map of paths defined in the input map[string]interface{} to the result of their validation
	// This is useful if you need more control
	fmt.Printf("Over %d fields\n", len(results.Fields))

	// You may be thinking that the validation above should have failed since there was an
	// extra key, 'count', defined that was encountered. By default lookslike does not
	// consider extra data to be an error. To change that behavior, wrap the validator
	// in lookslike.Strict()
	strictResults := Strict(validator)(data)

	fmt.Printf("Strict Results.Valid: %t\n", strictResults.Valid)

	// You can Check an exact field for an error
	fmt.Printf("For the count field specifically .Valid is: %t\n", strictResults.Fields["count"][0].Valid)

	// And get error objects for each error
	for _, err := range strictResults.Errors() {
		fmt.Println(err)
	}

	// And even get a new Results object with only invalid fields included
	strictResults.DetailedErrors()
}

func ExampleCompose() {
	// Composition is useful when you need to share common validation logic between validators.
	// Let's imagine that we want to validate maps describing pets.

	pets := []map[string]interface{}{
		{"Name": "rover", "barks": "often", "fur_length": "long"},
		{"Name": "lucky", "barks": "rarely", "fur_length": "short"},
		{"Name": "pounce", "meows": "often", "fur_length": "short"},
		{"Name": "peanut", "meows": "rarely", "fur_length": "long"},
	}

	// We can see that all pets have the "fur_length" property, but that only cats meow, and dogs bark.
	// We can concisely encode this in lookslike using lookslike.Compose.
	// We can also see that both "meows" and "barks" contain the same enums of values.
	// We'll start by creating a composed IsDef using the IsAny composition, which creates a new IsDef that is
	// a logical 'or' of its IsDef arguments

	isFrequency := isdef.IsAny(isdef.IsEqual("often"), isdef.IsEqual("rarely"))

	petValidator := MustCompile(map[string]interface{}{
		"Name":       isdef.IsNonEmptyString,
		"fur_length": isdef.IsAny(isdef.IsEqual("long"), isdef.IsEqual("short")),
	})
	dogValidator := Compose(
		petValidator,
		MustCompile(map[string]interface{}{"barks": isFrequency}),
	)
	catValidator := Compose(
		petValidator,
		MustCompile(map[string]interface{}{"meows": isFrequency}),
	)

	for _, pet := range pets {
		var petType string
		if dogValidator(pet).Valid {
			petType = "dog"
		} else if catValidator(pet).Valid {
			petType = "cat"
		}
		fmt.Printf("%s is a %s\n", pet["Name"], petType)
	}

	// Output:
	// rover is a dog
	// lucky is a dog
	// pounce is a cat
	// peanut is a cat
}

func ExampleOptional() {
	dataNoError := map[string]interface{}{"foo": "bar"}
	dataError := map[string]interface{}{"foo": "bar", "error": true}

	validator := MustCompile(map[string]interface{}{"foo": "bar", "error": isdef.Optional(isdef.IsEqual(true))})

	// Both inputs pass
	fmt.Printf("Validator classifies both maps as true: %t", validator(dataNoError).Valid && validator(dataError).Valid)

	// Output:
	// Validator classifies both maps as true: true
}

func ExampleIs() {
	// More advanced validations can be used with built-in and custom functions.
	// These are represented with the IfDef type

	data := map[string]interface{}{"foo": "bar", "count": 1}

	// Values can also be tested programatically if a lookslike.IsDef is used as a value
	// Here we'll define a custom IsDef using the lookslike DSL, then validate it.
	// The Is() function is the preferred way to costruct IsDef objects.
	startsWithB := isdef.Is("starts with b", func(path llpath.Path, v interface{}) *llresult.Results {
		vStr, ok := v.(string)
		if !ok {
			return llresult.SimpleResult(path, false, "Expected a string, got a %t", v)
		}

		if strings.HasPrefix(vStr, "b") {
			return llresult.ValidResult(path)
		}

		return llresult.SimpleResult(path, false, "Expected string to start with b, got %v", vStr)
	})

	funcValidator := MustCompile(map[string]interface{}{"foo": startsWithB})

	funcValidatorResult := funcValidator(data)

	fmt.Printf("Valid: %t", funcValidatorResult.Valid)

	// Output:
	// Valid: true
}

func ExampleMap() {
	v := MustCompile(map[string]interface{}{
		"foo": isdef.IsStringContaining("a"),
		"baz": "bot",
	})

	data := map[string]interface{}{
		"foo": "bar",
		"baz": "bot",
	}

	fmt.Printf("Result is %t", v(data).Valid)

	// Output:
	// Result is true
}

func ExampleSlice() {
	v := MustCompile(map[string]interface{}{
		"foo": []interface{}{"foo", isdef.IsNonEmptyString},
	})

	data := map[string]interface{}{"foo": []string{"foo", "something"}}

	fmt.Printf("Result is %t", v(data).Valid)

	// Output:
	// Result is true
}
