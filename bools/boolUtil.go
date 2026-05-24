package gw_bools

func CNullBoolByJson(json map[string]interface{}, key string) *bool {
	data := json[key]
	if data == nil {
		return nil
	}
	v, ok := data.(bool)
	if !ok {
		return nil
	}
	return &v
}
