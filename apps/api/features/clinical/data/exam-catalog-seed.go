// Package clinical_data holds the clinical seed content every tenant receives.
package clinical_data

import (
	"strings"
	"unicode"
)

// Exam categories (also the values stored on catalog items and order items).
const (
	ExamCategoryLaboratory = "laboratory"
	ExamCategoryImaging    = "imaging"
	ExamCategoryOther      = "other"
)

// SeedExam is one exam of the initial exam catalog.
type SeedExam struct {
	Name        string
	Category    string
	Subgroup    string
	Indications string
}

// SeedProfile is one profile of the initial exam catalog; Exams are exact
// SeedExam names.
type SeedProfile struct {
	Name  string
	Exams []string
}

// ExamSeedKey is the stable key of a seed exam or profile: it lets "restore
// defaults" find the seeded row even after the tenant renamed it. Never
// change a seed name without a migration for the rows already seeded.
func ExamSeedKey(name string) string {
	plain := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n").Replace(strings.ToLower(name))
	var b strings.Builder
	dash := false
	for _, r := range plain {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

func lab(subgroup string, exams ...[2]string) []SeedExam {
	return group(ExamCategoryLaboratory, subgroup, exams)
}

func img(subgroup string, exams ...[2]string) []SeedExam {
	return group(ExamCategoryImaging, subgroup, exams)
}

func group(category, subgroup string, exams [][2]string) []SeedExam {
	out := make([]SeedExam, 0, len(exams))
	for _, e := range exams {
		out = append(out, SeedExam{Name: e[0], Category: category, Subgroup: subgroup, Indications: e[1]})
	}
	return out
}

func concat(groups ...[]SeedExam) []SeedExam {
	var out []SeedExam
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

const (
	ayuno8        = "Ayuno de 8 horas."
	ayuno12       = "Ayuno de 12 horas."
	anticoag      = "Informar si toma anticoagulantes."
	cicloMenstrua = "Indicar el día del ciclo menstrual."
	psa           = "Evitar relaciones sexuales, ciclismo y equitación 48 horas antes."
	orinaEsteril  = "Primera orina de la mañana, chorro medio, previo aseo genital, en frasco estéril."
	primeraOrina  = "Primera orina de la mañana."
	antesAntib    = "Antes de iniciar antibióticos."
	ginecoPrep    = "No estar menstruando. Sin relaciones sexuales, duchas ni óvulos vaginales 48 horas antes."
	rxMetal       = "Retirar objetos metálicos. Informar si hay posibilidad de embarazo."
	rxEmbarazo    = "Informar si hay posibilidad de embarazo."
	vejigaLlena   = "Vejiga llena: tomar 1 litro de agua 1 hora antes y no orinar."
	metal         = "Retirar objetos metálicos."
	resonancia    = "Retirar objetos metálicos. Informar si tiene marcapasos, implantes o prótesis metálicas."
)

// ExamCatalogSeed is the initial exam catalog (docs/superpowers/specs/
// 2026-10-02-exam-catalog-seed.md), in display order.
var ExamCatalogSeed = concat(
	lab("Hematología",
		[2]string{"Biometría hemática completa", ""},
		[2]string{"Hemoglobina y hematocrito", ""},
		[2]string{"Recuento de plaquetas", ""},
		[2]string{"Recuento de reticulocitos", ""},
		[2]string{"Velocidad de sedimentación globular (VSG)", ""},
		[2]string{"Frotis de sangre periférica", ""},
		[2]string{"Grupo sanguíneo y factor Rh", ""},
	),
	lab("Coagulación",
		[2]string{"Tiempo de protrombina (TP) e INR", anticoag},
		[2]string{"Tiempo de tromboplastina parcial (TTP)", anticoag},
		[2]string{"Fibrinógeno", ""},
		[2]string{"Dímero D", ""},
		[2]string{"Tiempo de sangría y coagulación", ""},
	),
	lab("Química sanguínea",
		[2]string{"Glucosa en ayunas", ayuno8},
		[2]string{"Glucosa postprandial", "Tomar la muestra 2 horas después del desayuno habitual."},
		[2]string{"Curva de tolerancia a la glucosa", "Ayuno de 8 a 12 horas. Permanecer en reposo en el laboratorio durante la prueba (2 a 3 horas)."},
		[2]string{"Hemoglobina glicosilada (HbA1c)", ""},
		[2]string{"Urea", ""},
		[2]string{"Creatinina", ""},
		[2]string{"Ácido úrico", ayuno8},
		[2]string{"Colesterol total", ayuno12},
		[2]string{"Colesterol HDL", ayuno12},
		[2]string{"Colesterol LDL", ayuno12},
		[2]string{"Triglicéridos", "Ayuno de 12 horas. No consumir alcohol 24 horas antes."},
		[2]string{"Proteínas totales y fraccionadas", ""},
		[2]string{"Albúmina", ""},
		[2]string{"Bilirrubinas total y fraccionadas", ayuno8},
		[2]string{"TGO (AST)", ""},
		[2]string{"TGP (ALT)", ""},
		[2]string{"Fosfatasa alcalina", ayuno8},
		[2]string{"Gamma glutamil transferasa (GGT)", ""},
		[2]string{"Deshidrogenasa láctica (LDH)", ""},
		[2]string{"Amilasa", ""},
		[2]string{"Lipasa", ""},
		[2]string{"CPK total", "Evitar ejercicio intenso 48 horas antes."},
		[2]string{"CK-MB", ""},
		[2]string{"Troponina I", ""},
		[2]string{"NT-proBNP", ""},
		[2]string{"Insulina basal", ayuno8},
	),
	lab("Electrolitos y minerales",
		[2]string{"Sodio", ""},
		[2]string{"Potasio", ""},
		[2]string{"Cloro", ""},
		[2]string{"Calcio total", ""},
		[2]string{"Calcio iónico", ""},
		[2]string{"Magnesio", ""},
		[2]string{"Fósforo", ""},
	),
	lab("Hierro y vitaminas",
		[2]string{"Hierro sérico", "Ayuno de 8 horas, muestra en la mañana."},
		[2]string{"Ferritina", ""},
		[2]string{"Transferrina", ""},
		[2]string{"Capacidad total de fijación de hierro (TIBC)", ayuno8},
		[2]string{"Vitamina B12", ayuno8},
		[2]string{"Ácido fólico", ayuno8},
		[2]string{"Vitamina D (25-OH)", ""},
	),
	lab("Hormonas",
		[2]string{"TSH", ""},
		[2]string{"T4 libre", ""},
		[2]string{"T3 libre", ""},
		[2]string{"T4 total", ""},
		[2]string{"T3 total", ""},
		[2]string{"Anticuerpos anti-TPO", ""},
		[2]string{"Prolactina", "Muestra en la mañana, tras 20 minutos de reposo."},
		[2]string{"FSH", cicloMenstrua},
		[2]string{"LH", cicloMenstrua},
		[2]string{"Estradiol", cicloMenstrua},
		[2]string{"Progesterona", "Tomar la muestra el día 21 del ciclo, salvo otra indicación."},
		[2]string{"Testosterona total", "Muestra en la mañana."},
		[2]string{"DHEA-S", ""},
		[2]string{"Cortisol matutino", "Muestra entre las 7:00 y las 9:00."},
		[2]string{"Paratohormona (PTH)", ""},
		[2]string{"Beta-HCG cuantitativa", ""},
	),
	lab("Marcadores tumorales",
		[2]string{"PSA total", psa},
		[2]string{"PSA libre", psa},
		[2]string{"Antígeno carcinoembrionario (CEA)", ""},
		[2]string{"CA-125", "No tomar la muestra durante la menstruación."},
		[2]string{"CA 19-9", ""},
		[2]string{"CA 15-3", ""},
		[2]string{"Alfa-fetoproteína", ""},
	),
	lab("Inmunología y serología",
		[2]string{"Proteína C reactiva (PCR)", ""},
		[2]string{"Factor reumatoideo", ""},
		[2]string{"Antiestreptolisinas (ASTO)", ""},
		[2]string{"Anticuerpos antinucleares (ANA)", ""},
		[2]string{"Anticuerpos anti-CCP", ""},
		[2]string{"Complemento C3", ""},
		[2]string{"Complemento C4", ""},
		[2]string{"Procalcitonina", ""},
		[2]string{"VDRL", ""},
		[2]string{"VIH 1 y 2 (cuarta generación)", ""},
		[2]string{"Antígeno de superficie hepatitis B (HBsAg)", ""},
		[2]string{"Anticuerpos anti-HBs", ""},
		[2]string{"Anticuerpos hepatitis C (anti-HCV)", ""},
		[2]string{"Toxoplasma IgG e IgM", ""},
		[2]string{"Rubéola IgG e IgM", ""},
		[2]string{"Citomegalovirus IgG e IgM", ""},
		[2]string{"Epstein-Barr IgG e IgM", ""},
		[2]string{"Helicobacter pylori IgG", ""},
		[2]string{"Dengue NS1, IgG e IgM", ""},
		[2]string{"Reacciones febriles", ""},
		[2]string{"Prueba de embarazo cualitativa en sangre", ""},
		[2]string{"Antígeno SARS-CoV-2", ""},
		[2]string{"PCR SARS-CoV-2", ""},
	),
	lab("Orina",
		[2]string{"Elemental y microscópico de orina (EMO)", orinaEsteril},
		[2]string{"Urocultivo con antibiograma", orinaEsteril + " Antes de iniciar antibióticos."},
		[2]string{"Proteínas en orina de 24 horas", "Descartar la primera orina del día y recolectar toda la orina de las siguientes 24 horas, incluida la primera del día siguiente. Mantener refrigerada."},
		[2]string{"Depuración de creatinina en orina de 24 horas", "Recolección de orina de 24 horas (igual que proteínas en 24 h). Se toma también muestra de sangre."},
		[2]string{"Microalbuminuria", primeraOrina},
		[2]string{"Relación albúmina/creatinina en orina", primeraOrina},
	),
	lab("Heces",
		[2]string{"Coproparasitario", "Muestra fresca en frasco limpio; entregar dentro de 2 horas."},
		[2]string{"Coproparasitario seriado (3 muestras)", "Una muestra por día durante 3 días, en frascos separados."},
		[2]string{"Sangre oculta en heces", "Muestra fresca en frasco limpio."},
		[2]string{"Antígeno de Helicobacter pylori en heces", "Suspender inhibidores de bomba de protones 2 semanas y antibióticos 4 semanas antes, salvo otra indicación."},
		[2]string{"Coprocultivo", antesAntib},
		[2]string{"Rotavirus en heces", ""},
		[2]string{"Calprotectina fecal", ""},
		[2]string{"Polimorfonucleares en moco fecal", ""},
	),
	lab("Microbiología y citología",
		[2]string{"Cultivo y antibiograma de secreción (especificar sitio)", antesAntib},
		[2]string{"Hemocultivo", antesAntib},
		[2]string{"Baciloscopía (BK) en esputo, seriada", "Esputo de la primera tos de la mañana, 3 días seguidos, en frascos separados."},
		[2]string{"Cultivo de esputo", "Esputo de la primera tos de la mañana, previo enjuague bucal con agua."},
		[2]string{"Tinción de Gram (especificar muestra)", ""},
		[2]string{"KOH para hongos (especificar muestra)", "No aplicar cremas ni talcos en la zona 3 días antes."},
		[2]string{"Cultivo de secreción vaginal", "No tener relaciones sexuales ni usar duchas u óvulos vaginales 48 horas antes."},
		[2]string{"Citología cervicovaginal (Papanicolaou)", ginecoPrep},
		[2]string{"Detección de VPH", ginecoPrep},
	),
	img("Rayos X",
		[2]string{"Rx de tórax PA y lateral", rxMetal},
		[2]string{"Rx de tórax PA", rxMetal},
		[2]string{"Rx de columna cervical", rxEmbarazo},
		[2]string{"Rx de columna dorsal", rxEmbarazo},
		[2]string{"Rx de columna lumbosacra", rxEmbarazo},
		[2]string{"Rx de senos paranasales", ""},
		[2]string{"Rx de abdomen simple", rxEmbarazo},
		[2]string{"Rx de pelvis", rxEmbarazo},
		[2]string{"Rx de rodilla (especificar lado)", ""},
		[2]string{"Rx de mano y muñeca (especificar lado)", ""},
		[2]string{"Rx de hombro (especificar lado)", ""},
		[2]string{"Rx de tobillo y pie (especificar lado)", ""},
		[2]string{"Rx para edad ósea", ""},
	),
	img("Ecografía",
		[2]string{"Ecografía abdominal", ayuno8},
		[2]string{"Ecografía hepatobiliar", ayuno8},
		[2]string{"Ecografía renal y de vías urinarias", vejigaLlena},
		[2]string{"Ecografía pélvica", vejigaLlena},
		[2]string{"Ecografía transvaginal", "Vejiga vacía."},
		[2]string{"Ecografía obstétrica", ""},
		[2]string{"Ecografía vesical y prostática", vejigaLlena},
		[2]string{"Ecografía de tiroides", ""},
		[2]string{"Ecografía mamaria", ""},
		[2]string{"Ecografía de partes blandas (especificar zona)", ""},
		[2]string{"Ecografía testicular", ""},
		[2]string{"Doppler venoso de miembros inferiores", ""},
		[2]string{"Doppler arterial de miembros inferiores", ""},
		[2]string{"Doppler carotídeo", ""},
	),
	img("Mamografía y densitometría",
		[2]string{"Mamografía bilateral", "No usar desodorante, talco ni cremas ese día. Traer estudios previos."},
		[2]string{"Densitometría ósea", "No tomar suplementos de calcio 24 horas antes."},
	),
	img("Tomografía",
		[2]string{"Tomografía de cráneo simple", metal},
		[2]string{"Tomografía de tórax", metal},
		[2]string{"Tomografía abdominopélvica con contraste", "Ayuno de 6 horas. Traer creatinina reciente. Informar alergias y si es diabético."},
		[2]string{"Tomografía de senos paranasales", ""},
	),
	img("Resonancia magnética",
		[2]string{"Resonancia magnética de cerebro", resonancia},
		[2]string{"Resonancia magnética de columna lumbar", resonancia},
		[2]string{"Resonancia magnética de rodilla (especificar lado)", resonancia},
	),
	group(ExamCategoryOther, "", [][2]string{
		{"Electrocardiograma", ""},
		{"Ecocardiograma", ""},
		{"Prueba de esfuerzo", "Ropa cómoda y zapatos deportivos. Comida ligera 2 horas antes."},
		{"Holter de 24 horas", "Bañarse antes de la colocación; el equipo no debe mojarse."},
		{"Monitoreo ambulatorio de presión arterial (MAPA) 24 horas", "Ropa con mangas holgadas; el equipo no debe mojarse."},
		{"Espirometría", "No fumar 4 horas antes. Consultar si debe suspender broncodilatadores."},
		{"Electroencefalograma", "Cabello limpio y seco, sin cremas ni gel."},
		{"Electromiografía", "No aplicar cremas en la piel ese día."},
		{"Audiometría", "Evitar exposición a ruido intenso 12 horas antes."},
		{"Endoscopía digestiva alta", "Ayuno de 8 horas. Acudir con acompañante."},
		{"Colonoscopía", "Preparación intestinal según indicación médica. Acudir con acompañante."},
		{"Polisomnografía", "No consumir cafeína ni alcohol desde la tarde del estudio."},
		{"Fondo de ojo", "Acudir con acompañante; la dilatación pupilar puede dificultar conducir."},
	}),
)

// Names used by several profiles.
const (
	bhc        = "Biometría hemática completa"
	glucosa    = "Glucosa en ayunas"
	colTotal   = "Colesterol total"
	trigli     = "Triglicéridos"
	creat      = "Creatinina"
	acUrico    = "Ácido úrico"
	tgo        = "TGO (AST)"
	tgp        = "TGP (ALT)"
	emo        = "Elemental y microscópico de orina (EMO)"
	grupoRh    = "Grupo sanguíneo y factor Rh"
	vih        = "VIH 1 y 2 (cuarta generación)"
	pcr        = "Proteína C reactiva (PCR)"
	prolactina = "Prolactina"
)

// ExamProfileSeed is the initial set of profiles. A profile may mix
// categories (the pre-surgical one includes an electrocardiogram).
var ExamProfileSeed = []SeedProfile{
	{Name: "Chequeo general", Exams: []string{bhc, glucosa, colTotal, trigli, creat, acUrico, tgo, tgp, emo, "Coproparasitario"}},
	{Name: "Perfil lipídico", Exams: []string{colTotal, "Colesterol HDL", "Colesterol LDL", trigli}},
	{Name: "Perfil hepático", Exams: []string{tgo, tgp, "Fosfatasa alcalina", "Gamma glutamil transferasa (GGT)", "Bilirrubinas total y fraccionadas", "Proteínas totales y fraccionadas"}},
	{Name: "Perfil renal", Exams: []string{"Urea", creat, acUrico, emo}},
	{Name: "Perfil tiroideo", Exams: []string{"TSH", "T4 libre", "T3 libre"}},
	{Name: "Electrolitos", Exams: []string{"Sodio", "Potasio", "Cloro"}},
	{Name: "Control de diabetes", Exams: []string{glucosa, "Hemoglobina glicosilada (HbA1c)", creat, "Microalbuminuria", colTotal, trigli}},
	{Name: "Prequirúrgico", Exams: []string{bhc, grupoRh, "Tiempo de protrombina (TP) e INR", "Tiempo de tromboplastina parcial (TTP)", glucosa, creat, vih, emo, "Electrocardiograma"}},
	{Name: "Control prenatal (primer trimestre)", Exams: []string{bhc, grupoRh, glucosa, emo, "Urocultivo con antibiograma", "VDRL", vih, "Antígeno de superficie hepatitis B (HBsAg)", "Toxoplasma IgG e IgM", "Rubéola IgG e IgM", "Ecografía obstétrica"}},
	{Name: "Anemia", Exams: []string{bhc, "Recuento de reticulocitos", "Hierro sérico", "Ferritina", "Transferrina", "Vitamina B12", "Ácido fólico"}},
	{Name: "Síndrome febril", Exams: []string{bhc, pcr, emo, "Reacciones febriles", "Dengue NS1, IgG e IgM"}},
	{Name: "Perfil reumatológico", Exams: []string{"Velocidad de sedimentación globular (VSG)", pcr, "Factor reumatoideo", "Anticuerpos antinucleares (ANA)", "Anticuerpos anti-CCP", acUrico}},
	{Name: "Perfil hormonal femenino", Exams: []string{"FSH", "LH", "Estradiol", prolactina, "Progesterona"}},
}
