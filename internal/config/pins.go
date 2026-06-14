package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Pins struct {
	Projects []string `json:"projects"`
}

func pinsPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pins.json"), nil
}

func LoadPins() (*Pins, error) {
	path, err := pinsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Pins{}, nil
		}
		return nil, err
	}
	var p Pins
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func SavePins(p *Pins) error {
	if _, err := EnsureConfigDir(); err != nil {
		return err
	}
	path, err := pinsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func PinProject(name string) error {
	p, err := LoadPins()
	if err != nil {
		return err
	}
	for _, n := range p.Projects {
		if n == name {
			return nil
		}
	}
	p.Projects = append([]string{name}, p.Projects...)
	return SavePins(p)
}

func UnpinProject(name string) error {
	p, err := LoadPins()
	if err != nil {
		return err
	}
	var next []string
	for _, n := range p.Projects {
		if n != name {
			next = append(next, n)
		}
	}
	p.Projects = next
	return SavePins(p)
}

func IsPinned(name string) bool {
	p, err := LoadPins()
	if err != nil {
		return false
	}
	for _, n := range p.Projects {
		if n == name {
			return true
		}
	}
	return false
}

func SortProjectsPinnedFirst(names []string) []string {
	pins, _ := LoadPins()
	pinSet := map[string]bool{}
	for _, n := range pins.Projects {
		pinSet[n] = true
	}
	var pinned, rest []string
	for _, n := range names {
		if pinSet[n] {
			pinned = append(pinned, n)
		} else {
			rest = append(rest, n)
		}
	}
	out := append(pinned, rest...)
	return out
}
