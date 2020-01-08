package filters

import (
	"github.com/cbsinteractive/bakery/pkg/config"
	"github.com/cbsinteractive/bakery/pkg/parsers"
	"github.com/zencoder/go-dash/mpd"
)

// DASHFilter implements the Filter interface for DASH
// manifests
type DASHFilter struct {
	manifestURL     string
	manifestContent string
	config          config.Config
}

// NewDASHFilter is the DASH filter constructor
func NewDASHFilter(manifestURL, manifestContent string, c config.Config) *DASHFilter {
	return &DASHFilter{
		manifestURL:     manifestURL,
		manifestContent: manifestContent,
		config:          c,
	}
}

// FilterManifest will be responsible for filtering the manifest
// according  to the MediaFilters
func (d *DASHFilter) FilterManifest(filters *parsers.MediaFilters) (string, error) {
	manifest, err := mpd.ReadFromString(d.manifestContent)
	if err != nil {
		return "", err
	}

	if filters.CaptionTypes != nil {
		d.filterCaptionTypes(filters, manifest)
	}

	return manifest.WriteToString()
}

func (d *DASHFilter) filterCaptionTypes(filters *parsers.MediaFilters, manifest *mpd.MPD) {
	supportedTypes := map[parsers.CaptionType]bool{}

	for _, captionType := range filters.CaptionTypes {
		supportedTypes[captionType] = true
	}

	for _, period := range manifest.Periods {
		for _, as := range period.AdaptationSets {
			if as.ContentType == nil {
				continue
			}

			if *as.ContentType == "text" {
				var filteredReps []*mpd.Representation
				for _, r := range as.Representations {
					if r.Codecs == nil {
						filteredReps = append(filteredReps, r)
						continue
					}

					if _, supported := supportedTypes[parsers.CaptionType(*r.Codecs)]; supported {
						filteredReps = append(filteredReps, r)
					}
				}

				as.Representations = filteredReps
			}
		}
	}
}
