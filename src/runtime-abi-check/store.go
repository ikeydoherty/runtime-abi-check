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
	symbols map[elf.Machine]map[string]map[string]bool

	// Where we're allowed to look for system libraries.
	systemLibraries []string
}

// NewSymbolStore will return a newly setup symbol store..
func NewSymbolStore() *SymbolStore {
	ret := &SymbolStore{
		symbols: make(map[elf.Machine]map[string]map[string]bool),
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
func (s *SymbolStore) locateLibraryPaths(library string, inputFile *elf.File) []string {
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

// locateLibrary will attempt to find the right architecture library.
func (s *SymbolStore) locateLibrary(library string, inputFile *elf.File) (*elf.File, string, error) {
	possibles := s.locateLibraryPaths(library, inputFile)

	for _, p := range possibles {
		test, err := elf.Open(p)
		if err != nil {
			continue
		}
		if test.FileHeader.Machine != inputFile.FileHeader.Machine {
			fmt.Fprintf(os.Stderr, "Skipping incompatible library %s (%v)\n", p, test.FileHeader.Machine)
			test.Close()
			continue
		}
		fmt.Fprintf(os.Stderr, "Found library @ %v\n", p)
		return test, p, nil
	}
	return nil, "", fmt.Errorf("failed to locate: %v", library)
}

// ScanPath will attempt to scan an input file and work out symbol resolution
func (s *SymbolStore) ScanPath(path string) error {
	file, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	err = s.scanELF(path, file)
	if err != nil {
		return err
	}

	// Dump the symbol table
	for m := range s.symbols {
		fmt.Printf("Machine: %v\n\n", m)

		for lib := range s.symbols[m] {
			for sym := range s.symbols[m][lib] {
				fmt.Printf("%s:%s\n", lib, sym)
			}
		}
	}
	return nil
}

// scanELF is the internal recursion function to map out a symbol space completely
func (s *SymbolStore) scanELF(path string, file *elf.File) error {
	name := filepath.Base(path)

	// Figure out who we actually import
	libs, err := file.ImportedLibraries()
	if err != nil {
		return err
	}

	// Make sure we've got a bucket for the Machine
	if _, ok := s.symbols[file.FileHeader.Machine]; !ok {
		s.symbols[file.FileHeader.Machine] = make(map[string]map[string]bool)
	}

	// Find out what we actually expose..
	providesSymbols, err := file.DynamicSymbols()
	if err != nil {
		return err
	}

	if len(providesSymbols) > 0 {
		s.symbols[file.FileHeader.Machine][name] = make(map[string]bool)
	}

	for _, sym := range providesSymbols {
		// TODO: Filter symbols out if they're janky/weak
		// Store hit table
		s.symbols[file.FileHeader.Machine][name][sym.Name] = true
	}

	// At this point, we'd load all relevant libs
	for _, l := range libs {
		// Try and find the relevant guy. Basically, its an ELF and machine is matched
		lib, libPath, err := s.locateLibrary(l, file)
		if err != nil {
			return err
		}
		// Recurse into this Thing
		if err = s.scanELF(libPath, lib); err != nil {
			lib.Close()
			return err
		}
		lib.Close()
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
