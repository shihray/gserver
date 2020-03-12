package registry

type Service struct {
	Name     string            `json:"name"`
	ID       string            `json:"id"`
	Address  string            `json:"address"`
	Metadata map[string]string `json:"metadata"`
}
