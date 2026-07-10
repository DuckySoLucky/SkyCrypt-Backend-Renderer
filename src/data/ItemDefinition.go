package data

import (
	"fmt"

	"github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/global"
)

func ParseItemDefinitionJSON(data []byte) (map[string]interface{}, error) {
	var root interface{}
	if err := global.JSON.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	switch value := root.(type) {
	case map[string]interface{}:
		return value, nil
	case []interface{}:
		if len(value) == 0 {
			return nil, fmt.Errorf("item definition array is empty")
		}
		if len(value) != 1 {
			return nil, fmt.Errorf("item definition array must contain exactly one object, found %d entries", len(value))
		}
		definition, ok := value[0].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("item definition array entry must be an object, found %T", value[0])
		}
		return definition, nil
	default:
		return nil, fmt.Errorf("item definition root must be an object or singleton array, found %T", root)
	}
}
