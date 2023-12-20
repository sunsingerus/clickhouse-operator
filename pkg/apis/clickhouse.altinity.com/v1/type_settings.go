// Copyright 2019 Altinity Ltd and/or its affiliates. All rights reserved.
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

package v1

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/d4l3k/messagediff.v1"

	"github.com/altinity/clickhouse-operator/pkg/util"
)

// Specify returned errors for being re-used
var (
	errorNoSectionSpecified = fmt.Errorf("no section specified")
	errorNoPrefixSpecified  = fmt.Errorf("no prefix specified")
	errorNoSuffixSpecified  = fmt.Errorf("no suffix specified")
)

// Settings specifies settings
type Settings struct {
	m map[string]*Setting
}

// NewSettings creates new settings
func NewSettings() *Settings {
	m := makeSettings()
	return &m
}

func makeSettings() Settings {
	return Settings{
		m: make(map[string]*Setting),
	}
}

// Ensure ensures settings
func (s *Settings) Ensure() *Settings {
	if s == nil {
		return NewSettings()
	}
	return s
}

// Name2Key converts name to storage key. This is the opposite to Key2Name
func (s *Settings) Name2Key(name string) string {
	return name
}

// Key2Name converts storage key to name. This is the opposite to Name2Key
func (s *Settings) Key2Name(key string) string {
	return key
}

// Len gets length of the settings
func (s *Settings) Len() int {
	if s == nil {
		return 0
	}
	return len(s.m)
}

// WalkKeys walks over settings with a function. Function receives key and setting.
func (s *Settings) WalkKeys(f func(key string, setting *Setting)) {
	if s == nil {
		return
	}
	if s.Len() == 0 {
		return
	}
	for key := range s.m {
		f(key, s.GetKey(key))
	}
}

// Walk walks over settings with a function. Function receives name and setting.
// Storage key is used internally.
func (s *Settings) Walk(f func(name string, setting *Setting)) {
	if s == nil {
		return
	}
	if s.Len() == 0 {
		return
	}
	for key := range s.m {
		f(s.Key2Name(key), s.Get(s.Key2Name(key)))
	}
}

// HasKey checks whether key setting exists.
func (s *Settings) HasKey(key string) bool {
	if s == nil {
		return false
	}
	if s.Len() == 0 {
		return false
	}
	_, ok := s.m[key]
	return ok
}

// Has checks whether named setting exists.
// Storage key is used internally.
func (s *Settings) Has(name string) bool {
	if s == nil {
		return false
	}
	if s.Len() == 0 {
		return false
	}
	_, ok := s.m[s.Name2Key(name)]
	return ok
}

// GetKey gets key setting.
func (s *Settings) GetKey(key string) *Setting {
	if s == nil {
		return nil
	}
	if s.Len() == 0 {
		return nil
	}
	return s.m[key]
}

// Get gets named setting.
// Storage key is used internally.
func (s *Settings) Get(name string) *Setting {
	if s == nil {
		return nil
	}
	if s.Len() == 0 {
		return nil
	}
	return s.m[s.Name2Key(name)]
}

// SetKey sets key setting.
func (s *Settings) SetKey(key string, setting *Setting) *Settings {
	if s == nil {
		return s
	}
	s.m[key] = setting
	return s
}

// Set sets named setting.
// Storage key is used internally.
func (s *Settings) Set(name string, setting *Setting) *Settings {
	if s == nil {
		return s
	}
	s.m[s.Name2Key(name)] = setting
	return s
}

// DeleteKey deletes key setting
func (s *Settings) DeleteKey(key string) {
	if s == nil {
		return
	}
	if !s.HasKey(key) {
		return
	}
	// Delete storage key
	delete(s.m, key)
}

// Delete deletes named setting
func (s *Settings) Delete(name string) {
	if s == nil {
		return
	}
	if !s.Has(name) {
		return
	}
	// Delete storage key
	delete(s.m, s.Name2Key(name))
}

// IsZero checks whether settings is zero
func (s *Settings) IsZero() bool {
	if s == nil {
		return true
	}
	return s.Len() == 0
}

// SetIfNotExists sets named setting
func (s *Settings) SetIfNotExists(name string, setting *Setting) *Settings {
	if s == nil {
		return s
	}
	if !s.Has(name) {
		s.Set(name, setting)
	}

	return s
}

// SetScalarsFromMap sets multiple scalars from map
func (s *Settings) SetScalarsFromMap(m map[string]string) *Settings {
	// Copy values from the map
	for name, value := range m {
		s.Set(name, NewSettingScalar(value))
	}

	return s
}

// Keys gets keys of the settings
func (s *Settings) Keys() (keys []string) {
	s.WalkKeys(func(key string, setting *Setting) {
		keys = append(keys, key)
	})
	return keys
}

// Names gets names of the settings
func (s *Settings) Names() (names []string) {
	s.Walk(func(name string, setting *Setting) {
		names = append(names, name)
	})
	return names
}

// UnmarshalJSON unmarshal JSON
func (s *Settings) UnmarshalJSON(data []byte) error {
	if s == nil {
		return fmt.Errorf("unable to unmashal with nil")
	}

	// Prepare untyped map at first
	type untypedMapType map[string]any
	var untypedMap untypedMapType

	// Provided binary data is expected to unmarshal into untyped map, because settings are map-like struct
	if err := json.Unmarshal(data, &untypedMap); err != nil {
		return err
	}

	// Entries are expected to exist
	if len(untypedMap) == 0 {
		return nil
	}

	// Create entries from untyped map in result settings
	for key, untyped := range untypedMap {
		if scalarSetting, ok := NewSettingScalarFromAny(untyped); ok {
			s.SetKey(key, scalarSetting)
			continue // for
		}

		if vectorSetting, ok := NewSettingVectorFromAny(untyped); ok {
			if vectorSetting.Len() > 0 {
				s.SetKey(key, vectorSetting)
			}
			continue // for
		}

		// Unknown type of entry in untyped map
		// Should error be reported?
		// Skip for now
	}

	return nil
}

// MarshalJSON marshals JSON
func (s *Settings) MarshalJSON() ([]byte, error) {
	if s == nil {
		return json.Marshal(nil)
	}

	raw := make(map[string]interface{})
	s.WalkKeys(func(key string, setting *Setting) {
		raw[key] = setting.AsAny()
	})

	return json.Marshal(raw)
}

// fetchPort
func (s *Settings) fetchPort(name string) int32 {
	return int32(s.Get(name).ScalarInt())
}

// GetTCPPort gets TCP port from settings
func (s *Settings) GetTCPPort() int32 {
	return s.fetchPort("tcp_port")
}

// GetTCPPortSecure gets TCP port secure from settings
func (s *Settings) GetTCPPortSecure() int32 {
	return s.fetchPort("tcp_port_secure")
}

// GetHTTPPort gets HTTP port from settings
func (s *Settings) GetHTTPPort() int32 {
	return s.fetchPort("http_port")
}

// GetHTTPSPort gets HTTPS port from settings
func (s *Settings) GetHTTPSPort() int32 {
	return s.fetchPort("https_port")
}

// GetInterserverHTTPPort gets interserver HTTP port from settings
func (s *Settings) GetInterserverHTTPPort() int32 {
	return s.fetchPort("interserver_http_port")
}

// MergeFrom merges into `dst` non-empty new-key-values from `src` in case no such `key` already in `src`
func (s *Settings) MergeFrom(src *Settings) *Settings {
	if src.Len() == 0 {
		return s
	}

	src.Walk(func(name string, value *Setting) {
		s = s.Ensure().SetIfNotExists(name, value)
	})

	return s
}

// MergeFromCB merges settings from src approved by filtering callback function
func (s *Settings) MergeFromCB(src *Settings, filter func(name string, setting *Setting) bool) *Settings {
	if src.Len() == 0 {
		return s
	}

	src.Walk(func(name string, value *Setting) {
		if filter(name, value) {
			// Accept this setting
			s = s.Ensure().Set(name, value)
		}
	})

	return s
}

// ExtractSection returns map of the specified settings section
func (s *Settings) ExtractSection(section SettingsSection, includeSettingWithNoSectionSpecified bool) (m map[string]string) {
	if s == nil {
		return nil
	}

	s.WalkKeys(func(key string, setting *Setting) {
		_section, err := getSectionFromPath(key)
		if (err == nil) && !_section.Equal(section) {
			// This is not the section we are looking for, skip to the next
			return
		}
		if (err != nil) && (err != errorNoSectionSpecified) {
			// We have a complex error, skip to the next
			return
		}
		if (err == errorNoSectionSpecified) && !includeSettingWithNoSectionSpecified {
			// We are not ready to include setting with unspecified section, skip to the next
			return
		}

		// Looks like we are ready to include this setting into result set

		filename, err := getFilenameFromPath(key)
		if err != nil {
			// We need to have filename specified
			return
		}

		if !setting.IsScalar() {
			// We are ready to accept scalars only
			return
		}

		if m == nil {
			// Lazy load
			m = make(map[string]string)
		}

		// Fetch file content
		m[filename] = setting.ScalarString()
	})

	return m
}

// Filter filters settings according to include and exclude lists
func (s *Settings) Filter(
	includeSections []SettingsSection,
	excludeSections []SettingsSection,
	includeSettingWithNoSectionSpecified bool,
) (res *Settings) {
	if s.Len() == 0 {
		return res
	}

	s.WalkKeys(func(key string, _ *Setting) {
		section, err := getSectionFromPath(key)

		if (err != nil) && (err != errorNoSectionSpecified) {
			// We have a complex error, skip to the next
			return
		}
		if (err == errorNoSectionSpecified) && !includeSettingWithNoSectionSpecified {
			// We are not ready to include unspecified section, skip to the next
			return
		}

		// No include sections specified is treated as 'include by default'
		include := section.In(includeSections) || (includeSections == nil)
		exclude := section.In(excludeSections)

		if !include || exclude {
			// This is not the section we are looking for, skip to the next
			return
		}

		// We'd like to get this setting
		res = res.Ensure().SetKey(key, s.GetKey(key))
	})

	return res
}

// AsSortedSliceOfStrings return settings as sorted strings
func (s *Settings) AsSortedSliceOfStrings() []string {
	if s == nil {
		return nil
	}

	// Sort keys
	var keys []string
	s.WalkKeys(func(key string, _ *Setting) {
		keys = append(keys, key)
	})
	sort.Strings(keys)

	var res []string

	// Walk over sorted keys
	for _, key := range keys {
		res = append(res, key)
		res = append(res, s.GetKey(key).String())
	}

	return res
}

// Normalize normalizes settings
func (s *Settings) Normalize() *Settings {
	s.normalizeKeys()
	return s
}

// normalizeKeys normalizes keys in settings, treating them as paths
func (s *Settings) normalizeKeys() {
	if s.Len() == 0 {
		return
	}

	var keysToNormalize []string

	// Find entries with keys to normalize
	s.WalkKeys(func(key string, _ *Setting) {
		if _, modified := normalizeKeyAsPath(key); modified {
			// Normalization changed something. This path has to be normalized
			keysToNormalize = append(keysToNormalize, key)
		}
	})

	// Add entries with normalized keys
	for _, unNormalizedKey := range keysToNormalize {
		normalizedKey, _ := normalizeKeyAsPath(unNormalizedKey)
		s.SetKey(normalizedKey, s.GetKey(unNormalizedKey))
	}

	// Delete entries with un-normalized keys
	for _, unNormalizedKey := range keysToNormalize {
		s.DeleteKey(unNormalizedKey)
	}
}

// normalizeKeyAsPath normalizes key which is treated as a path
// Normalized key looks like 'a/b/c'
// Used in in .spec.configuration.{users, profiles, quotas, settings, files} sections
func normalizeKeyAsPath(path string) (string, bool) {
	// Find all multi-'/' values (like '//')
	re := regexp.MustCompile("//+")

	// Squash all multi-'/' values (like '//') to single-'/'
	normalized := re.ReplaceAllString(path, "/")
	// Cut all leading and trailing '/', so the result would be 'a/b/c'
	normalized = strings.Trim(normalized, "/")

	return normalized, normalized != path
}

func getPrefixFromPath(path string) (string, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		// We need to have path to be at least 2 entries in order to have prefix
		return "", errorNoPrefixSpecified
	}

	// Extract the first component from the path
	prefix := parts[0]
	if prefix == "" {
		return "", errorNoPrefixSpecified
	}

	return prefix, nil
}

func getSuffixFromPath(path string) (string, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		// We need to have path to be at least one entry - which will be the suffix
		return "", errorNoSuffixSpecified
	}

	// Extract the last component from the path
	suffix := parts[len(parts)-1]
	if suffix == "" {
		// We need to have path to be at least one entry - which will be the suffix
		return "", errorNoSuffixSpecified
	}

	return suffix, nil
}

// getSectionFromPath
func getSectionFromPath(path string) (SettingsSection, error) {
	// String representation of the section
	section, err := getPrefixFromPath(path)
	if err != nil {
		// We need to have path to be at least 'section/file.name'
		return SectionEmpty, errorNoSectionSpecified
	}

	// Check dir names to determine which section path points to
	configDir := section
	switch {
	case strings.EqualFold(configDir, CommonConfigDir):
		return SectionCommon, nil
	case strings.EqualFold(configDir, UsersConfigDir):
		return SectionUsers, nil
	case strings.EqualFold(configDir, HostConfigDir):
		return SectionHost, nil
	}

	{
		// TODO - either provide example or just remove this part
		// Check explicitly specified sections. This is never(?) used
		section := NewSettingsSectionFromString(section)
		switch {
		case SectionCommon.Equal(section):
			return SectionCommon, nil
		case SectionUsers.Equal(section):
			return SectionUsers, nil
		case SectionHost.Equal(section):
			return SectionHost, nil
		}
	}

	return SectionEmpty, fmt.Errorf("unknown section specified %v", section)
}

func getFilenameFromPath(path string) (string, error) {
	return getSuffixFromPath(path)
}

// listModifiedSettingsPaths makes list of paths that were modified between two settings.
// Ex.:
// confid.d/setting1
// confid.d/setting2
func listModifiedSettingsPaths(a, b *Settings, path *messagediff.Path, value interface{}) (paths []string) {
	if settings, ok := (value).(*Settings); ok {
		// Provided `value` is of type api.Settings, which means that the whole
		// settings such as 'files' or 'settings' is being either added or removed
		if settings == nil {
			// Completely removed settings such as 'files' or 'settings', so the value changed from Settings to nil
			// List all entries from settings that are removed
			for _, name := range a.Keys() {
				paths = append(paths, name)
			}
		} else {
			// Introduced new settings such as 'files' or 'settings', so the value changed from nil to Settings
			// List all entries from settings that is added
			for _, name := range b.Keys() {
				paths = append(paths, name)
			}
		}
	} else {
		// Provided `value` is not of type api.Settings, thus expecting it to be a piece of settings.
		// Modify (without full removal or addition) settings such as 'files' or 'settings',
		// something is still left in the remaining part of settings in case of deletion or added in case of addition.
		// Build string representation of path to updated element
		var pathElements []string
		for _, pathNode := range *path {
			switch mk := pathNode.(type) {
			case messagediff.MapKey:
				switch pathElement := mk.Key.(type) {
				case string:
					pathElements = append(pathElements, pathElement)
				}
			}
		}
		paths = append(paths, strings.Join(pathElements, "/"))
	}

	return paths
}

// listPrefixedModifiedSettingsPaths makes list of paths that were modified between two settings.
// Each entry in the list is prefixed with the specified `pathPrefix`
// Ex.: `prefix` = file
// file/setting1
// file/setting2
func listPrefixedModifiedSettingsPaths(a, b *Settings, pathPrefix string, path *messagediff.Path, value interface{}) (paths []string) {
	return util.Prefix(listModifiedSettingsPaths(a, b, path, value), pathPrefix+"/")
}

// ListAffectedSettingsPathsFromDiff makes list of paths that were modified between two settings prefixed with the specified `prefix`
// Ex.: `prefix` = file
// file/setting1
// file/setting2
func ListAffectedSettingsPathsFromDiff(a, b *Settings, diff *messagediff.Diff, prefix string) (paths []string) {
	for path, value := range diff.Added {
		paths = append(paths, listPrefixedModifiedSettingsPaths(a, b, prefix, path, value)...)
	}
	for path, value := range diff.Removed {
		paths = append(paths, listPrefixedModifiedSettingsPaths(a, b, prefix, path, value)...)
	}
	for path, value := range diff.Modified {
		paths = append(paths, listPrefixedModifiedSettingsPaths(a, b, prefix, path, value)...)
	}
	return paths
}
