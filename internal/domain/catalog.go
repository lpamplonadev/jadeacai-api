package domain

type CatalogCategory string

const (
	CatalogCategoryCombo   CatalogCategory = "combo"
	CatalogCategoryGourmet CatalogCategory = "gourmet"
)

func IsCatalogCategory(category string) bool {
	switch CatalogCategory(category) {
	case CatalogCategoryCombo, CatalogCategoryGourmet:
		return true
	default:
		return false
	}
}

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
