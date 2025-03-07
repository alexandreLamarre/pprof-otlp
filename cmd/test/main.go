package main

import (
	"os"

	pprofotlp "github.com/alexandreLamarre/pprof-otlp"
	colprofilespb "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"

	profilespb "go.opentelemetry.io/proto/otlp/profiles/v1development"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	toOTLPTEST()

	// toSVGTEST()
}

func toSVGTEST() {
	file := "./testdata/otlp/profiles.json"

	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}

	var dest colprofilespb.ExportProfilesServiceRequest
	if err := protojson.Unmarshal(data, &dest); err != nil {
		panic(err)
	}
	pprofotlp.ToSvg(dest.ResourceProfiles[0])

}

func toOTLPTEST() {
	file := "./testdata/pprof/profile.pb"
	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	profile, err := pprofotlp.ParseBytes(data)
	if err != nil {
		panic(err)
	}

	otlpData, err := pprofotlp.ToOTLP(profile)
	if err != nil {
		panic(err)
	}
	pprofotlp.ToSvg(&profilespb.ResourceProfiles{
		ScopeProfiles: []*profilespb.ScopeProfiles{
			{
				Profiles: []*profilespb.Profile{
					otlpData,
				},
			},
		},
	})
}
