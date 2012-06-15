// Copyright 2012 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

// This file contains the model construction by reflection.

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/dsymonds/gomock/mockgen/model"
)

func Reflect(importPath, symbol string) (*model.Package, error) {
	// TODO: sanity check arguments

	// We use TempDir instead of TempFile so we can control the filename.
	tmpDir, err := ioutil.TempDir("", "gomock_reflect_")
	if err != nil {
		return nil, err
	}
	defer func() { os.RemoveAll(tmpDir) }()
	const progSource = "prog.go"

	// Generate program.
	var program bytes.Buffer
	data := reflectData{
		ImportPath: importPath,
		Symbol:     symbol,
	}
	if err := reflectProgram.Execute(&program, &data); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, progSource), program.Bytes(), 0600); err != nil {
		return nil, err
	}

	// Run it.
	cmd := exec.Command("go", "run", progSource)
	cmd.Dir = tmpDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Process output.
	var pkg model.Package
	if err := gob.NewDecoder(&stdout).Decode(&pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

type reflectData struct {
	ImportPath string
	Symbol     string
}

// This program reflects on an interface value, and prints the
// gob encoding of a model.Package to standard output.
// JSON doesn't work because of the model.Type interface.
var reflectProgram = template.Must(template.New("program").Parse(`
package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"reflect"

	"github.com/dsymonds/gomock/mockgen/model"

	pkg_ {{printf "%q" .ImportPath}}
)

func main() {
	it := reflect.TypeOf((*pkg_.{{.Symbol}})(nil)).Elem()
	pkg, err := model.PackageFromInterfaceType(it)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Reflection: %v\n", err)
		os.Exit(1)
	}
	if err := gob.NewEncoder(os.Stdout).Encode(pkg); err != nil {
		fmt.Fprintf(os.Stderr, "gob encode: %v\n", err)
		os.Exit(1)
	}
}
`))