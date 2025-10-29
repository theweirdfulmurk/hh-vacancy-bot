package models

var ExperienceMapping = map[string]string{
	"Нет опыта":     "noExperience",
	"От 1 до 3 лет": "between1And3",
	"От 3 до 6 лет": "between3And6",
	"Более 6 лет":   "moreThan6",
}

var ExperienceDisplayNames = map[string]string{
	"noExperience": "Нет опыта",
	"between1And3": "От 1 до 3 лет",
	"between3And6": "От 3 до 6 лет",
	"moreThan6":    "Более 6 лет",
}

var ScheduleMapping = map[string]string{
	"Полный день":      "fullDay",
	"Сменный график":   "shift",
	"Гибкий график":    "flexible",
	"Удаленная работа": "remote",
	"Вахтовый метод":   "flyInFlyOut",
}

var ScheduleDisplayNames = map[string]string{
	"fullDay":     "Полный день",
	"shift":       "Сменный график",
	"flexible":    "Гибкий график",
	"remote":      "Удаленная работа",
	"flyInFlyOut": "Вахтовый метод",
}

func ExperienceOptions() []string {
	return []string{
		"Нет опыта",
		"От 1 до 3 лет",
		"От 3 до 6 лет",
		"Более 6 лет",
	}
}

func ScheduleOptions() []string {
	return []string{
		"Полный день",
		"Сменный график",
		"Гибкий график",
		"Удаленная работа",
		"Вахтовый метод",
	}
}

func IsValidExperience(text string) bool {
	_, ok := ExperienceMapping[text]
	return ok
}

func IsValidSchedule(text string) bool {
	_, ok := ScheduleMapping[text]
	return ok
}

func GetExperienceID(displayName string) string {
	if id, ok := ExperienceMapping[displayName]; ok {
		return id
	}
	return ""
}

func GetScheduleID(displayName string) string {
	if id, ok := ScheduleMapping[displayName]; ok {
		return id
	}
	return ""
}

func GetExperienceDisplayName(id string) string {
	if name, ok := ExperienceDisplayNames[id]; ok {
		return name
	}
	return id
}

func GetScheduleDisplayName(id string) string {
	if name, ok := ScheduleDisplayNames[id]; ok {
		return name
	}
	return id
}