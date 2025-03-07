package pprofotlp

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/pprof/profile"
	"github.com/samber/lo"

	// colprofilepb "go.opentelemetry.io/proto/otlp/collector/profiles/v1"
	v11 "go.opentelemetry.io/proto/otlp/common/v1"

	profilespb "go.opentelemetry.io/proto/otlp/profiles/v1development"
)

func ParseBytes(data []byte) (*profile.Profile, error) {
	r := bytes.NewReader(data)
	return profile.Parse(r)
}

func PprintFunction(f *profile.Function) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("| FunctionId: %d |", f.ID))
	sb.WriteString(fmt.Sprintf(" Name: %s |", f.Name))
	sb.WriteString(fmt.Sprintf(" SystemName: %s |", f.SystemName))
	sb.WriteString(fmt.Sprintf(" Filename: %s |", f.Filename))
	sb.WriteString(fmt.Sprintf(" StartLine: %d |", f.StartLine))

	return sb.String()
}

func PprintMapping(m *profile.Mapping) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("| ID: %d |", m.ID))
	sb.WriteString(fmt.Sprintf(" Start: %d |", m.Start))
	sb.WriteString(fmt.Sprintf(" Limit: %d |", m.Limit))
	sb.WriteString(fmt.Sprintf(" Offset: %d |", m.Offset))
	sb.WriteString(fmt.Sprintf(" File: %s |", m.File))
	sb.WriteString(fmt.Sprintf(" HasFunctions: %t |", m.HasFunctions))
	sb.WriteString(fmt.Sprintf(" HasFilenames: %t |", m.HasFilenames))
	sb.WriteString(fmt.Sprintf(" HasLineNumbers: %t |", m.HasLineNumbers))
	sb.WriteString(fmt.Sprintf(" HasInlineFrames: %t |", m.HasInlineFrames))
	return sb.String()
}

func PprintSample(s *profile.Sample) string {
	sb := strings.Builder{}
	fmt.Println(s.Value)
	sb.WriteString(fmt.Sprintf("| Label: %v |\n", s.Label))
	// sb.WriteString(fmt.Sprintf("| Location: %d |",))
	// if len(s.Location) != len(s.Value) {
	// 	panic("unexpected mismatch?")
	// }
	prefix := ""
	for i, l := range s.Location {
		sb.WriteString(fmt.Sprintf("%s [%d] : %s |\n", prefix, i, PprintLocation(l)))
		prefix += "\t"
	}

	return sb.String()
}

func PprintLocation(l *profile.Location) string {
	retLine := ""
	for _, line := range l.Line {
		retLine += PprintLine(line) + " "
	}
	return fmt.Sprintf("ID: %d | Address: %d | Mapping: %d | IsFolded: %t | Line: %s", l.ID, l.Address, l.Mapping.ID, l.IsFolded, retLine)
}

func PprintLine(l profile.Line) string {
	return fmt.Sprintf("--Function: %d | Line: %d | Column : %d--", l.Function.ID, l.Line, l.Column)
}

const defaultKey = "root"

// groups by equality
func groupByAttributes(
	samples []*profilespb.Sample,
	attrTable []*v11.KeyValue,
	keys ...string,

) map[string][]*profilespb.Sample {
	if len(keys) == 0 {
		panic("invalid : no keys to group by")
	}
	ret := map[string][]*profilespb.Sample{}
	for _, sample := range samples {

		// key
		key := ""
		slices.Sort(sample.AttributeIndices)
		for _, attrI := range sample.AttributeIndices {
			if slices.Contains(keys, attrTable[attrI].Key) {
				if key != "" {
					key += "/"
				}
				key += string(attrTable[attrI].Value.String())
			}
		}
		if key == "" {
			key = defaultKey
		}
		sl, ok := ret[key]
		if !ok {
			ret[key] = []*profilespb.Sample{}
			ret[key] = append(ret[key], sample)
		} else {
			ret[key] = append(sl, sample)
		}
	}

	for mKey, mValue := range ret {
		fmt.Println("Key : ", mKey, "Values : ", len(mValue))
	}

	return ret
}

func ToSvg(profile *profilespb.ResourceProfiles) ([]byte, error) {

	for _, scope := range profile.ScopeProfiles {
		for _, p := range scope.Profiles {
			// Example Unix nano timestamp
			startNano := uint64(p.TimeNanos) // Replace with your timestamp
			// Convert to time.Time
			startTime := time.Unix(0, int64(startNano))
			durationNano := int64(p.DurationNanos)
			durationTime := time.Unix(0, durationNano)

			// Pretty-print the timestamp
			fmt.Printf("Start time: %s\n", startTime.Format(time.RFC3339Nano))
			if durationNano != 0 {
				fmt.Printf("Duration time: %s", durationTime.Format(time.RFC3339Nano))
			}

			fmt.Println("Period : ", p.Period)
			groupedSamples := groupByAttributes(p.Sample, p.AttributeTable, "thread.name", "process.pid")

			for key, samples := range groupedSamples {
				fmt.Printf("======== %s =======\n", key)
				fmt.Printf("====== Samples : %d ======\n", len(samples))
				for _, s := range samples {
					durs := []time.Duration{}
					for _, t := range s.TimestampsUnixNano {
						unixNano := time.Unix(0, int64(t))
						durs = append(durs, startTime.Sub(unixNano))
					}
					strDur := []string{}
					for _, dur := range durs {
						strDur = append(strDur, dur.String())
					}
					fmt.Println("Durations : [ ", strings.Join(strDur, ","), " ]")
					entryPoint := s.LocationsStartIndex
					entryLocations := p.LocationTable[entryPoint]
					mapping := p.MappingTable[*entryLocations.MappingIndex]
					fmt.Println(mapping.AttributeIndices)

					locs := p.LocationIndices[s.LocationsStartIndex : s.LocationsStartIndex+s.LocationsLength]
					prefix := ""
					for _, loc := range locs {
						loca := p.LocationTable[loc]
						for _, line := range loca.Line {
							f := p.FunctionTable[line.FunctionIndex]
							fmt.Println(prefix, p.StringTable[f.NameStrindex])
							prefix += "  "
						}
						prefix += "  "
					}
				}
			}
			// panic("done")

			for _, s := range p.Sample {
				fmt.Println("======== sample ======= ", s.Value)
				for _, attrI := range s.AttributeIndices {
					fmt.Println(string(p.AttributeTable[attrI].Key), " : ", string(p.AttributeTable[attrI].Value.String()))
				}
				fmt.Println(s.LocationsLength)
				fmt.Println(s.TimestampsUnixNano)
				durs := []time.Duration{}
				for _, t := range s.TimestampsUnixNano {
					unixNano := time.Unix(0, int64(t))
					durs = append(durs, startTime.Sub(unixNano))
				}
				strDur := []string{}
				for _, dur := range durs {
					strDur = append(strDur, dur.String())
				}
				fmt.Println("Durations : [ ", strings.Join(strDur, ","), " ]")
				entryPoint := s.LocationsStartIndex
				entryLocations := p.LocationTable[entryPoint]
				mapping := p.MappingTable[*entryLocations.MappingIndex]
				fmt.Println(mapping.AttributeIndices)

				locs := p.LocationIndices[s.LocationsStartIndex : s.LocationsStartIndex+s.LocationsLength]
				prefix := ""
				for _, loc := range locs {
					loca := p.LocationTable[loc]
					for _, line := range loca.Line {
						f := p.FunctionTable[line.FunctionIndex]
						fmt.Println(prefix, p.StringTable[f.NameStrindex])
						prefix += "  "
					}
					prefix += "  "
				}
			}

		}
	}
	return nil, nil
}

func functionConverter(workingProfile *profilespb.Profile, function *profile.Function) *profilespb.Function {
	nameIdx := 0
	systemNameIdx := 0
	fileNameIdx := 0
	for i, v := range workingProfile.StringTable {
		if v == function.Name {
			nameIdx = i
		}
		if v == function.SystemName {
			systemNameIdx = i
		}
		if v == function.Filename {
			fileNameIdx = i
		}
	}
	if nameIdx == 0 && function.Name != "" {
		workingProfile.StringTable = append(workingProfile.StringTable, function.Name)
		nameIdx = len(workingProfile.StringTable) - 1
	}
	if systemNameIdx == 0 && function.SystemName != "" {
		workingProfile.StringTable = append(workingProfile.StringTable, function.SystemName)
		systemNameIdx = len(workingProfile.StringTable) - 1
	}
	if fileNameIdx == 0 && function.Filename != "" {
		workingProfile.StringTable = append(workingProfile.StringTable, function.Filename)
		fileNameIdx = len(workingProfile.StringTable) - 1
	}

	return &profilespb.Function{
		NameStrindex:       int32(nameIdx),
		SystemNameStrindex: int32(systemNameIdx),
		FilenameStrindex:   int32(fileNameIdx),
		StartLine:          function.StartLine,
	}
}

func mappingCoverter(workingProfile *profilespb.Profile, mapping *profile.Mapping) *profilespb.Mapping {
	fileIdx := 0
	for i, v := range workingProfile.StringTable {
		if v == mapping.File {
			fileIdx = i
		}
	}
	if fileIdx == 0 && mapping.File != "" {
		workingProfile.StringTable = append(workingProfile.StringTable, mapping.File)
		fileIdx = len(workingProfile.StringTable) - 1
	}

	return &profilespb.Mapping{
		HasFunctions:     mapping.HasFunctions,
		HasFilenames:     mapping.HasFilenames,
		HasLineNumbers:   mapping.HasLineNumbers,
		HasInlineFrames:  mapping.HasInlineFrames,
		MemoryStart:      mapping.Start,
		MemoryLimit:      mapping.Limit,
		FileOffset:       mapping.Offset,
		AttributeIndices: []int32{},
		FilenameStrindex: int32(fileIdx),
	}
}

func locationConverter(
	workingProfile *profilespb.Profile,
	location *profile.Location,
) *profilespb.Location {
	mapping := location.Mapping
	mappingIdx := 0
	for i, m := range workingProfile.MappingTable {
		if m.MemoryStart == mapping.Start && m.MemoryLimit == mapping.Limit && m.FileOffset == mapping.Offset {
			mappingIdx = i
		}
	}

	newLines := []*profilespb.Line{}
	for _, line := range location.Line {
		newLines = append(newLines, lineConverter(workingProfile, line))
	}

	return &profilespb.Location{
		AttributeIndices: []int32{},
		Line:             newLines,
		MappingIndex:     lo.ToPtr(int32(mappingIdx)),
		Address:          location.Address,
		IsFolded:         location.IsFolded,
	}
}

func lineConverter(workingProfile *profilespb.Profile, location profile.Line) *profilespb.Line {
	fIdx := 0
	for i, f := range workingProfile.FunctionTable {
		fName := workingProfile.StringTable[f.NameStrindex]
		if fName == location.Function.Name {
			fIdx = i
		}
	}

	if fIdx == 0 && location.Function != nil {
		workingProfile.FunctionTable = append(
			workingProfile.FunctionTable,
			functionConverter(workingProfile, location.Function),
		)
		fIdx = len(workingProfile.FunctionTable) - 1
	}

	return &profilespb.Line{
		FunctionIndex: int32(fIdx),
		Line:          location.Line,
		Column:        location.Column,
	}
}

func sampleConverter(
	workingProfile *profilespb.Profile,
	sample *profile.Sample,
) *profilespb.Sample {
	locations := sample.Location
	startIdx := len(workingProfile.LocationTable)
	if len(locations) == 0 {
		panic("unexpected?")
	}
	for _, loc := range locations {
		workingProfile.LocationTable = append(
			workingProfile.LocationTable,
			locationConverter(workingProfile, loc),
		)
		workingProfile.LocationIndices = append(workingProfile.LocationIndices, int32(len(workingProfile.LocationTable)-1))
	}
	attrIds := []int32{}
	for k, v := range sample.Label {
		workingProfile.AttributeTable = append(
			workingProfile.AttributeTable,
			&v11.KeyValue{
				Key:   k,
				Value: &v11.AnyValue{Value: &v11.AnyValue_StringValue{StringValue: strings.Join(v, ",")}},
			},
		)
		attrIds = append(attrIds, int32(len(workingProfile.AttributeTable)-1))
	}

	return &profilespb.Sample{
		Value:               sample.Value,
		LocationsStartIndex: int32(startIdx),
		LocationsLength:     int32(len(locations)),
		AttributeIndices:    attrIds,

		// not sure how to populate this yet
		TimestampsUnixNano: []uint64{},

		// not sure what this does yet
		LinkIndex: nil,
	}
}

func ToOTLP(profile *profile.Profile) (*profilespb.Profile, error) {

	functions := profile.Function
	mappings := profile.Mapping

	workingProfile := &profilespb.Profile{
		SampleType:      []*profilespb.ValueType{},
		Sample:          []*profilespb.Sample{},
		MappingTable:    []*profilespb.Mapping{},
		LocationTable:   []*profilespb.Location{},
		LocationIndices: []int32{},
		FunctionTable:   []*profilespb.Function{},
		AttributeTable:  []*v11.KeyValue{},
		AttributeUnits:  []*profilespb.AttributeUnit{},
		LinkTable:       []*profilespb.Link{},
		StringTable: []string{
			"",
		},
	}
	for _, f := range functions {
		workingProfile.FunctionTable = append(
			workingProfile.FunctionTable,
			functionConverter(workingProfile, f),
		)
	}

	fmt.Println(workingProfile.StringTable)
	fmt.Println(workingProfile.FunctionTable)

	for _, m := range mappings {
		workingProfile.MappingTable = append(
			workingProfile.MappingTable,
			mappingCoverter(workingProfile, m),
		)
	}

	fmt.Println(workingProfile.MappingTable)

	for _, sample := range profile.Sample {
		workingProfile.Sample = append(
			workingProfile.Sample,
			sampleConverter(
				workingProfile,
				sample,
			),
		)
	}
	return workingProfile, nil
}
