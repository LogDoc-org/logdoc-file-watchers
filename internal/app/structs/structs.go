package structs

type Config struct {
	Debug  bool   `json:"debug"`
	LogDoc LD     `json:"logdoc"`
	Files  []File `json:"files"`
}

type LD struct {
	Host    string  `json:"host"`
	Port    string  `json:"port"`
	Proto   string  `json:"proto"`
	Default Default `json:"default"`
}

type File struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
	App     string `json:"app"`
	Source  string `json:"source"`
	Level   string `json:"level"`
	Layout  string `json:"layout"`
	Custom  string `json:"custom"`
}

type Default struct {
	App    string `json:"app"`
	Source string `json:"source"`
	Level  string `json:"level"`
}
