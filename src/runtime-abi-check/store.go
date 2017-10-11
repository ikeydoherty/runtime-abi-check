//
// Copyright © 2017 Ikey Doherty
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"debug/elf"
	"fmt"
)

// SymbolStore is used to create a global mapping so that we can resolve symbols
// within a process space
type SymbolStore struct {
	// symbols map Machine -> library name -> symbol
	// TODO: Consider making this full library path to symbol and resolve that way..
	symbols map[elf.Machine]map[string]string
}

// NewSymbolStore will return a newly setup symbol store..
func NewSymbolStore() *SymbolStore {
	ret := &SymbolStore{
		symbols: make(map[elf.Machine]map[string]string),
	}
	return ret
}

// ScanPath will attempt to scan an input file and work out symbol resolution
func (s *SymbolStore) ScanPath(path string) error {
	file, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Figure out who we actually import
	libs, err := file.ImportedLibraries()
	if err != nil {
		return err
	}

	// At this point, we'd load all relevant libs
	fmt.Println(libs)

	// Figure out what symbols we end up using
	syms, err := file.ImportedSymbols()
	if err != nil {
		return err
	}

	// At this point, we'd resolve all symbols..
	// The "Library" may actually be empty, so we need to go looking through
	// a symbol store for this process to find out who actually owns it
	for _, sym := range syms {
		if sym.Library != "" {
			continue
		}
		fmt.Printf("Resolve symbol: %v\n", sym.Name)
	}

	return nil
}
