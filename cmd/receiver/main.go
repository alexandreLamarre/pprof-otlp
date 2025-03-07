package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	colprofilespb "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func handler(w http.ResponseWriter, r *http.Request) {
	var dest colprofilespb.ExportProfilesServiceRequest
	data, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := proto.Unmarshal(data, &dest); err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err = protojson.Marshal(&dest)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	os.WriteFile("profiles.json", data, 0644)
	w.WriteHeader(http.StatusOK)
	os.Exit(1)
}
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1development/profiles", handler)
	srv := &http.Server{
		Addr:    "127.0.0.1:7007",
		Handler: mux,
	}
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
