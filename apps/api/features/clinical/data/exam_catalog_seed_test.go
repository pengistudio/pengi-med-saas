package clinical_data

import "testing"

func TestExamCatalogSeed_MatchesApprovedContent(t *testing.T) {
	counts := map[string]int{}
	keys := map[string]bool{}
	for _, exam := range ExamCatalogSeed {
		counts[exam.Category]++
		key := ExamSeedKey(exam.Name)
		if key == "" || keys[key] {
			t.Errorf("seed key %q of %q is empty or duplicated", key, exam.Name)
		}
		keys[key] = true
		if exam.Category != ExamCategoryOther && exam.Subgroup == "" {
			t.Errorf("%q has no subgroup", exam.Name)
		}
	}
	want := map[string]int{ExamCategoryLaboratory: 121, ExamCategoryImaging: 36, ExamCategoryOther: 13}
	for category, n := range want {
		if counts[category] != n {
			t.Errorf("%s: got %d exams, want %d", category, counts[category], n)
		}
	}
	if len(ExamCatalogSeed) != 170 {
		t.Errorf("got %d exams, want 170", len(ExamCatalogSeed))
	}

	if len(ExamProfileSeed) != 13 {
		t.Errorf("got %d profiles, want 13", len(ExamProfileSeed))
	}
	profileKeys := map[string]bool{}
	for _, profile := range ExamProfileSeed {
		key := ExamSeedKey(profile.Name)
		if profileKeys[key] {
			t.Errorf("duplicated profile key %q", key)
		}
		profileKeys[key] = true
		for _, name := range profile.Exams {
			if !keys[ExamSeedKey(name)] {
				t.Errorf("profile %q references unknown exam %q", profile.Name, name)
			}
		}
	}
}

func TestExamSeedKey(t *testing.T) {
	cases := map[string]string{
		"Ácido úrico":                      "acido-urico",
		"Tiempo de protrombina (TP) e INR": "tiempo-de-protrombina-tp-e-inr",
		"Dengue NS1, IgG e IgM":            "dengue-ns1-igg-e-igm",
		"Relación albúmina/creatinina":     "relacion-albumina-creatinina",
	}
	for in, want := range cases {
		if got := ExamSeedKey(in); got != want {
			t.Errorf("ExamSeedKey(%q) = %q, want %q", in, got, want)
		}
	}
}
