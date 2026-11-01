package instagram

func gqlWidth(m *Media) int32 {
	if m == nil || m.Dimensions == nil {
		return 0
	}
	return m.Dimensions.Width
}
func gqlHeight(m *Media) int32 {
	if m == nil || m.Dimensions == nil {
		return 0
	}
	return m.Dimensions.Height
}
