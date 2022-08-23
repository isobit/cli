package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Env interface {
	Lookup(key string) (value string, ok bool)
}

type OSEnv struct {}
func (OSEnv) Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

type MapEnv struct {
	Data map[string]string
}
func NewMapEnv(data map[string]string) MapEnv {
	return MapEnv{Data: data}
}
func (me MapEnv) Lookup(key string) (string, bool) {
	value, ok := me.Data[key]
	return value, ok
}


type EnvFile struct {
	data map[string]string
}
func (ef EnvFile) Lookup(key string) (string, bool) {
	value, ok := ef.data[key]
	return value, ok
}
func ParseEnvFile(path string) (*EnvFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data := map[string]string{}
	scanner := bufio.NewScanner(file)
	for i := 1; scanner.Scan(); i++ {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		kv := strings.SplitN(line, "=", 1)
		if len(kv) != 2 {
			return nil, fmt.Errorf("error on line %d: not of form KEY=VAL", i)
		}
		data[kv[0]] = kv[1]
	}
	return &EnvFile{data}, nil
}
