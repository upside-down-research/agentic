package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"upside-down-research.com/oss/agentic/internal/llm"
)

func TestPlanning(t *testing.T) {
	p := PlanCollection{
		Plans: []Plan{
			{
				Name:       "Plan 1",
				SystemType: "Planner",
				Rationale:  "To plan",
				Definition: PlanDefinition{
					Inputs: []InOut{
						{
							Name: "Input 1",
							Type: "String",
						},
					},
					Outputs: []InOut{
						{
							Name: "Output 1",
							Type: "Integer",
						},
					},
					Behavior: "To plan",
				},
			},
		},
	}
	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(bytes))
}

func TestCodeDefinition_Unmarshal(t *testing.T) {
	ip := &ImplementedPlan{}
	j := `{
     "environment": "local",
     "coding_language": "Go",
     "code": [
       {
         "filename": "web_server.go",
         "content": "package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n\nfunc initializeWebServer() {\n\thttp.HandleFunc(\"/\", func(w http.ResponseWriter, r *http.Request) {\n\t\tfmt.Fprintf(w, \"Welcome to the home page!\")\n\t})\n\thttp.HandleFunc(\"/post\", func(w http.ResponseWriter, r *http.Request) {\n\t\tif r.Method == http.MethodPost {\n\t\t\tfmt.Fprintf(w, \"POST request successful\")\n\t\t} else {\n\t\t\tw.WriteHeader(http.StatusMethodNotAllowed)\n\t\t\tfmt.Fprintf(w, \"Only POST requests are allowed on this endpoint\")\n\t\t}\n\t})\n\n\tfmt.Println(\"Starting server at http://localhost:8080\")\n\terr := http.ListenAndServe(\":8080\", nil)\n\tif err != nil {\n\t\tfmt.Println(\"Error starting server: \", err)\n\t}\n}\n"
       }
     ]
   }
`
	err := json.Unmarshal([]byte(j), &ip)
	if err != nil {
		t.Fatal(err)
	}

}

func TestNewRun(t *testing.T) {
	type args struct {
		runID      string
		outputPath string
	}
	tests := []struct {
		name string
		args args
		want *Run
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRun(tt.args.runID, tt.args.outputPath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRun_AnswerAndVerify(t *testing.T) {
	type fields struct {
		RunID      string
		OutputPath string
		RunRecords map[int]RunRecord
		latestRun  int
		Mutex      sync.Mutex
	}
	type args struct {
		s           llm.Server
		query       string
		finalOutput any
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &Run{
				RunID:      tt.fields.RunID,
				OutputPath: tt.fields.OutputPath,
				RunRecords: tt.fields.RunRecords,
				latestRun:  tt.fields.latestRun,
				Mutex:      tt.fields.Mutex,
			}
			got, err := run.AnswerAndVerify(tt.args.s, tt.args.query, tt.args.finalOutput)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnswerAndVerify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AnswerAndVerify() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRun_AppendRecord(t *testing.T) {
	type fields struct {
		RunID      string
		OutputPath string
		RunRecords map[int]RunRecord
		latestRun  int
		Mutex      sync.Mutex
	}
	type args struct {
		query  string
		answer string
		takes  []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &Run{
				RunID:      tt.fields.RunID,
				OutputPath: tt.fields.OutputPath,
				RunRecords: tt.fields.RunRecords,
				latestRun:  tt.fields.latestRun,
				Mutex:      tt.fields.Mutex,
			}
			run.AppendRecord(tt.args.query, tt.args.answer, tt.args.takes)
		})
	}
}

func TestRun_WriteData(t *testing.T) {
	type fields struct {
		RunID      string
		OutputPath string
		RunRecords map[int]RunRecord
		latestRun  int
		Mutex      sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &Run{
				RunID:      tt.fields.RunID,
				OutputPath: tt.fields.OutputPath,
				RunRecords: tt.fields.RunRecords,
				latestRun:  tt.fields.latestRun,
				Mutex:      tt.fields.Mutex,
			}
			run.WriteData()
		})
	}
}

func TestStringPrompt(t *testing.T) {
	type args struct {
		label string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringPrompt(tt.args.label); got != tt.want {
				t.Errorf("StringPrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}
