package console

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestAcademyRoutesAndCurriculum(t *testing.T) {
	h := NewHandler()
	for _, path := range []string{"/console/academy", "/console/academy/", "/api/curriculum"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Fatalf("%s: empty or failed response", path)
		}
	}
	var curriculum struct {
		Modules []struct {
			ID            string   `json:"id"`
			Prerequisites []string `json:"prerequisites"`
			Options       []string `json:"options"`
			Answer        int      `json:"answer"`
			Lab           string   `json:"lab"`
			Lesson        string   `json:"lesson"`
			Exercise      []string `json:"exercise"`
		} `json:"modules"`
		Projects []struct {
			Modules []string `json:"modules"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(CurriculumJSON, &curriculum); err != nil {
		t.Fatal(err)
	}
	if len(curriculum.Modules) < 20 || len(curriculum.Projects) < 3 {
		t.Fatal("curriculum incomplete")
	}
	seen := map[string]bool{}
	labs := map[string]bool{}
	for _, lab := range GetAllLabs() {
		labs[lab.ID] = true
	}
	for _, m := range curriculum.Modules {
		if m.ID == "" || seen[m.ID] {
			t.Fatalf("invalid/duplicate ID %q", m.ID)
		}
		for _, p := range m.Prerequisites {
			if !seen[p] {
				t.Fatalf("%s prerequisite %s is missing or out of order", m.ID, p)
			}
		}
		if m.Answer < 0 || m.Answer >= len(m.Options) || m.Lesson == "" || len(m.Exercise) == 0 {
			t.Fatalf("invalid lesson %s", m.ID)
		}
		if m.Lab != "" && !labs[m.Lab] {
			t.Fatalf("missing lab %s", m.Lab)
		}
		seen[m.ID] = true
	}
	for _, p := range curriculum.Projects {
		for _, id := range p.Modules {
			if !seen[id] {
				t.Fatalf("project references missing module %s", id)
			}
		}
	}
}

func TestUnverifiedLabsDoNotPass(t *testing.T) {
	for _, id := range []string{"unknown", "lab-s3-website", "lab-iam-least-privilege", "lab-sqs-decoupling", "lab-dynamodb-crud"} {
		result := CheckLab(id)
		if result.Passed || result.Score != 0 || len(result.Details) == 0 {
			t.Fatalf("%s falsely verified: %+v", id, result)
		}
	}
}
