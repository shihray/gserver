package module_type

type ModuleTypeName string

const (
	Ping ModuleTypeName = "PING"
	Pong ModuleTypeName = "PONG"
)

func (typeName ModuleTypeName) String() string {
	return string(typeName)
}
