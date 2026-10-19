package preferences

type AssetOrder string

const (
	OrderAsc  AssetOrder = "asc"
	OrderDesc AssetOrder = "desc"
)

func (AssetOrder) Enum() []string {
	return []string{string(OrderAsc), string(OrderDesc)}
}

type PersonUpdateStrategy string

const (
	UpdateSelf     PersonUpdateStrategy = "self"
	UpdateEveryone PersonUpdateStrategy = "everyone"
)

func (PersonUpdateStrategy) Enum() []string {
	return []string{string(UpdateSelf), string(UpdateEveryone)}
}
