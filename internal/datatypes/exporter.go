package datatypes

type IExporter interface {
	ToJSON() ([]byte, error)

	ToTOML() ([]byte, error)
}
