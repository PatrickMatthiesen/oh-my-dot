package manifest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLocalOptionsOverlayAndDisabledPresence(t *testing.T) {
	base, err := ParseManifestReader(strings.NewReader(`{"features":[{"name":"prompt","disabled":true,"options":{"theme":"shared","config_file":"shared-file","keep":true}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	local, err := ParseManifestReader(strings.NewReader(`{"features":[{"name":"prompt","options":{"theme":"private","config_file":null}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	merged := MergeManifests(base, local).Features[0]
	if !merged.Disabled {
		t.Fatal("options-only local override re-enabled base feature")
	}
	if merged.Options["theme"] != "private" || merged.Options["keep"] != true {
		t.Fatalf("options not overlaid: %+v", merged.Options)
	}
	if _, exists := merged.Options["config_file"]; exists {
		t.Fatal("local tombstone did not remove inherited option")
	}
	if base.Features[0].Options["theme"] != "shared" || base.Features[0].Options["config_file"] != "shared-file" {
		t.Fatal("merge mutated base options")
	}
	local.Features[0].Disabled = false
	local.Features[0].DisabledSet = true
	data, err := json.Marshal(local)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"disabled":false`) {
		t.Fatal("explicit local false was omitted")
	}
	roundtrip, err := ParseManifestReader(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	if !roundtrip.Features[0].DisabledSet || MergeManifests(base, roundtrip).Features[0].Disabled {
		t.Fatal("explicit local enable was not preserved")
	}
}

func TestLocalOnlyFeatureOrderIsStable(t *testing.T) {
	local := &FeatureManifest{Features: []FeatureConfig{{Name: "first"}, {Name: "second"}}}
	for count := 0; count < 25; count++ {
		merged := MergeManifests(&FeatureManifest{}, local)
		if merged.Features[0].Name != "first" || merged.Features[1].Name != "second" {
			t.Fatal("local manifest order changed")
		}
	}
}

func TestLocalOnlyTombstoneHasNoEffectiveValue(t *testing.T) {
	local := &FeatureManifest{Features: []FeatureConfig{{Name: "prompt", Options: map[string]any{"config_file": nil, "theme": "pure"}}}}
	merged := MergeManifests(&FeatureManifest{}, local)
	if _, exists := merged.Features[0].Options["config_file"]; exists {
		t.Fatal("local-only tombstone remained an effective option")
	}
	if _, exists := local.Features[0].Options["config_file"]; !exists {
		t.Fatal("merge mutated the local manifest")
	}
}

func TestDisabledNullDoesNotEnableInheritedFeature(t *testing.T) {
	if _, err := ParseManifestReader(strings.NewReader(`{"features":[{"name":"prompt","disabled":null}]}`)); err == nil {
		t.Fatal("null disabled value was accepted as an explicit enable")
	}
}
