/*
Some code in this file was copied from the go "flag" package source and
modified. That code's license is retained here:

Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package cli

import (
	"fmt"
)

type parser struct {
	fields map[string]field
	parsed bool
	args   []string
}

func (p *parser) parse(arguments []string) error {
	p.parsed = true
	p.args = arguments
	for {
		seen, err := p.parseOne()
		if err != nil {
			return err
		}
		if !seen {
			break
		}
	}
	return nil
}

func (p *parser) parseOne() (bool, error) {
	if len(p.args) == 0 {
		return false, nil
	}
	s := p.args[0]
	if len(s) < 2 || s[0] != '-' {
		return false, nil
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			p.args = p.args[1:]
			return false, nil
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, fmt.Errorf("bad flag syntax: %s", s)
	}

	// If single dash, handle each rune in the name as a separate flag, except
	// for the last one which can be handled normally since it make have a
	// following argument.
	if numMinuses == 1 {
		i := 0
		for ; i < len(name)-1; i++ {
			shortName := name[i]
			if err := p.parseOneFlag(string(shortName), false, "", false); err != nil {
				return false, err
			}
		}
		name = name[i:]
	}

	// it's a flag. does it have an argument?
	p.args = p.args[1:]
	hasValue := false
	value := ""
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}

	if err := p.parseOneFlag(name, hasValue, value, true); err != nil {
		return false, err
	}

	return true, nil
}

func (p *parser) parseOneFlag(name string, hasValue bool, value string, canLookNext bool) error {
	field, ok := p.fields[name]
	if !ok {
		return fmt.Errorf("flag provided but not defined: %s", name)
	}

	fv := field.value

	if fv.isBoolFlag { // special case: doesn't need an arg
		if hasValue {
			if err := fv.Set(value); err != nil {
				return fmt.Errorf("invalid boolean value %q for flag %s: %v", value, name, err)
			}
		} else {
			if err := fv.Set("true"); err != nil {
				return fmt.Errorf("invalid boolean flag %s: %v", name, err)
			}
		}
	} else {
		// It must have a value, which might be the next argument.
		if !hasValue && len(p.args) > 0 && canLookNext {
			// value is the next arg
			hasValue = true
			value, p.args = p.args[0], p.args[1:]
		}
		if !hasValue {
			return fmt.Errorf("flag needs an argument: %s", name)
		}
		if err := fv.Set(value); err != nil {
			return fmt.Errorf("invalid value %q for flag %s: %v", value, name, err)
		}
	}
	return nil
}
