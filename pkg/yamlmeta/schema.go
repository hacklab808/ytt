// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package yamlmeta

import (
	"fmt"
)

type Schema interface {
	AssignType(node *Node)
}

type AnySchema struct {
}

type DocumentSchema struct {
	Allowed *DocumentType
}

type TypeCheck struct {
	Violations []string
}

func (tc *TypeCheck) HasViolations() bool {
	return len(tc.Violations) > 0
}

type DocumentType struct {
	*Document
}

type MapType struct {
	Map
}

type MapItemType struct {
	MapItem
}

type ArrayType struct {
	Array
}

func NewDocumentSchema(doc *Document) (*DocumentSchema, error) {
	schemaDoc := &DocumentSchema{Allowed: &DocumentType{&Document{Value: nil}}}

	switch typedContent := doc.Value.(type) {
	case *Map:
		mapType, err := NewMapSchema(typedContent)
		if err != nil {
			return nil, err
		}
		schemaDoc.Allowed = &DocumentType{&Document{Value: mapType}}
	case *Array:
		return nil, NewArraySchema()
	}

	return schemaDoc, nil
}

func NewMapSchema(m *Map) (*MapType, error) {
	mapType := &MapType{}
	for _, mapItem := range m.Items {
		newMapItem, err := NewMapItemSchema(mapItem)
		if err != nil {
			return nil, err
		}
		mapType.Items = append(mapType.Items, newMapItem)
	}
	return mapType, nil
}

func NewMapItemSchema(item *MapItem) (*MapItem, error) {
	switch typedContent := item.Value.(type) {
	case *Map:
		newMap, err := NewMapSchema(typedContent)
		if err != nil {
			return nil, err
		}
		return &MapItem{Key: item.Key, Value: newMap, Type: MapType{}}, nil
	case string:
		return &MapItem{Key: item.Key, Value: item.Value, Type: "string"}, nil
	case int:
		return &MapItem{Key: item.Key, Value: item.Value, Type: "scalar"}, nil
	case *Array:
		return nil, NewArraySchema()
	}
	return nil, fmt.Errorf("Map Item type did not match any know types")
}

func NewArraySchema() error {
	return fmt.Errorf("Arrays are currently not supported in schema")
}

//func NewStringSchema(str string) {
//
//}
//
//func NewScalarSchema(num float64) {
//
//}

func (mt *MapType) AllowsKey(key interface{}) bool {
	for _, item := range mt.Items {
		if item.Key == key {
			return true
		}
	}
	return false
}

func (mt MapType) CheckAllows(item *MapItem) TypeCheck {
	typeCheck := TypeCheck{}

	if !mt.AllowsKey(item.Key) {
		typeCheck.Violations = append(typeCheck.Violations, fmt.Sprintf("Map item '%s' at %s is not defined in schema", item.Key, item.Position.AsCompactString()))
	}
	return typeCheck
}

func (d *Document) Check() TypeCheck {
	var typeCheck TypeCheck

	switch typedContents := d.Value.(type) {
	case Node:
		typeCheck = typedContents.Check()
	}

	return typeCheck
}

func (m *Map) Check() TypeCheck {
	typeCheck := TypeCheck{}

	for _, item := range m.Items {
		check := m.Type.CheckAllows(item)
		if check.HasViolations() {
			typeCheck.Violations = append(typeCheck.Violations, check.Violations...)
			continue
		}
		check = item.Check()
		if check.HasViolations() {
			typeCheck.Violations = append(typeCheck.Violations, check.Violations...)
		}
	}
	return typeCheck
}

func (mapItem *MapItem) Check() TypeCheck {
	typeCheck := TypeCheck{}

	//mapItem.Type.CheckAllows()

	//switch t := mapItem.Value.(type) {
	//case :
	//}
	return typeCheck
}

func (d *DocumentSet) Check() TypeCheck { return TypeCheck{} }
func (d *Array) Check() TypeCheck       { return TypeCheck{} }
func (d *ArrayItem) Check() TypeCheck   { return TypeCheck{} }

func (as AnySchema) AssignType(_ *Node) {
	//(*node). = DocumentType{}
}

func (s DocumentSchema) AssignType(node *Node) {
	vals := (*node).GetValues()
	for _, val := range vals {
		switch typedNode := val.(type) {
		case *Map:
			mapType, ok := s.Allowed.Value.(*MapType)
			if !ok {
				typedNode.Type = &MapType{}
				// during typing we dont report error
				break
			}
			typedNode.Type = mapType

			for _, item := range typedNode.Items {
				for _, mapTypeItem := range mapType.Items {
					if item.Key == mapTypeItem.Key {
						item.Type = mapTypeItem.Type

						_, ok := s.Allowed.Value.(*MapType)
						if ok {
							oldAllowed := s.Allowed
							s.Allowed = &DocumentType{&Document{Value: mapTypeItem.Value}}
							s.AssignType(item.Value.(*Node))
							s.Allowed = oldAllowed
						}

					}
				}
			}
		case *MapItem:
			mapType, ok := s.Allowed.Value.(*MapType)
			if !ok {
				typedNode.Type = &MapType{}
				// during typing we dont report error
				break
			}
			typedNode.Type = mapType

			for _, mapTypeItem := range mapType.Items {
				if typedNode.Key == mapTypeItem.Key {
					typedNode.Type = mapTypeItem.Type

					_, ok := s.Allowed.Value.(*MapType)
					if ok {
						oldAllowed := s.Allowed
						s.Allowed = &DocumentType{&Document{Value: mapTypeItem.Value}}
						s.AssignType(typedNode.Value.(*Node))
						s.Allowed = oldAllowed
					}

				}
			}
		}
	}
}
