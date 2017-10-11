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
	"os"
	"path/filepath"
)

// SymbolStore is used to create a global mapping so that we can resolve symbols
// within a process space
type SymbolStore struct {
	// symbols map Machine -> library name -> symbol
	// TODO: Consider making this full library path to symbol and resolve that way..
	symbols map[elf.Machine]map[string]string

	// Where we're allowed to look for system libraries.
	systemLibraries []string
}

// NewSymbolStore will return a newly setup symbol store..
func NewSymbolStore() *SymbolStore {
	ret := &SymbolStore{
		symbols: make(map[elf.Machine]map[string]string),
		// Typical set of paths known by linux distributions
		systemLibraries: []string{
			"/usr/lib64",
			"/usr/lib",
			"/usr/lib/x86_64-linux-gnu",
			"/usr/lib/i386-linux-gnu",
			"/usr/lib32",
		},
	}

	return ret
}

// locateLibrary is a private method to determine where a library might actually
// be found on the system
func (s *SymbolStore) locateLibrary(library string, inputFile *elf.File) []string {
	var ret []string
	var searchPath []string
	// TODO: Be unstupid and accept DT_RUNPATH foo as well as faked LD_LIBRARY_PATH
	searchPath = append(searchPath, s.systemLibraries...)
	for _, p := range searchPath {
		// Find out if the guy exists.
		fullPath := filepath.Join(p, library)
		st, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		// Using stat not lstat..
		if !st.Mode().IsRegular() {
			continue
		}
		ret = append(ret, fullPath)

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
	for _, l := range libs {
		possibles := s.locateLibrary(l, file)
		fmt.Println(possibles)
	}

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
