package domain

type CatalogItemKind string

const (
	CatalogFlavor            CatalogItemKind = "flavor"
	CatalogSize              CatalogItemKind = "size"
	CatalogTopping           CatalogItemKind = "topping"
	CatalogSauce             CatalogItemKind = "sauce"
	CatalogCondimentPosition CatalogItemKind = "condiment_position"
	CatalogFruit             CatalogItemKind = "fruit"
	CatalogExtra             CatalogItemKind = "extra"
)

func IsCatalogItemKind(kind string) bool {
	switch CatalogItemKind(kind) {
	case CatalogFlavor, CatalogSize, CatalogTopping, CatalogSauce, CatalogCondimentPosition, CatalogFruit, CatalogExtra:
		return true
	default:
		return false
	}
}
