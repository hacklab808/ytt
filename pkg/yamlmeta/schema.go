// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package yamlmeta

import (
	"fmt"
)

type Schema interface {
	AssignType(node Node)
}

type AnySchema struct {
}

var _ Schema = &AnySchema{}

type DocumentSchema struct {
	Allowed *DocumentType
}

var _ Schema = &DocumentSchema{}

type MapSchema struct {
	Allowed *MapType
}

var _ Schema = &MapSchema{}

type MapItemSchema struct {
	Allowed interface{}
}

var _ Schema = &MapItemSchema{}

type ValueSchema struct {
	Allowed ValueType
}

func (v ValueSchema) AssignType(node Node) {
	panic("implement me")
}

var _ Schema = &ValueSchema{}

type TypeCheck struct {
	Violations []string
}

func (tc *TypeCheck) HasViolations() bool {
	return len(tc.Violations) > 0
}

type DocumentType struct {
	Schema interface{}
}

type ValueType struct {
	DefaultValue interface{}
}

type MapType struct {
	Items    []*MapItemType
}

type MapItemType struct {
	Key          interface{}
	Schema       Schema
}

type ArrayType struct {
	Array
}

func NewDocumentSchema(doc *Document) (*DocumentSchema, error) {
	schemaDoc := &DocumentSchema{Allowed: &DocumentType{&Document{Value: nil}}}

	switch typedContent := doc.Value.(type) {
	case *Map:
		mapSchema, err := NewMapSchema(typedContent)
		if err != nil {
			return nil, err
		}
		schemaDoc.Allowed.Schema = mapSchema
	case *Array:
		return nil, NewArraySchema()
	}

	return schemaDoc, nil
}

func NewMapSchema(m *Map) (*MapSchema, error) {
	mapType := &MapType{}

	for _, mapItem := range m.Items {
		newMapItemSchema, err := NewMapItemType(mapItem)
		if err != nil {
			return nil, err
		}
		mapType.Items = append(mapType.Items, newMapItemSchema)
	}
	return &MapSchema{Allowed: mapType}, nil
}

func NewMapItemType(schemaItem *MapItem) (*MapItemType, error) {
	switch typedContent := schemaItem.Value.(type) {
	case *Map:
		newMapSchema, err := NewMapSchema(typedContent)
		if err != nil {
			return nil, err
		}
		return &MapItemType{Key: schemaItem.Key, Schema: newMapSchema}, nil
	case string:
		return &MapItemType{Key: schemaItem.Key, Schema: ValueSchema{Allowed: ValueType{DefaultValue: schemaItem.Value}}}, nil
	case int:
		return &MapItemType{Key: schemaItem.Key, Schema: ValueSchema{Allowed: ValueType{DefaultValue: schemaItem.Value}}}, nil
	case *Array:
		return nil, NewArraySchema()
	}
	return nil, fmt.Errorf("Map Item type did not match any know types")
}

func NewArraySchema() error {
	return fmt.Errorf("Arrays are currently not supported in schema")
}

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
	violationErrorMessage := "Map item '%s' at %s was type %T when %T was expected."

	switch typedValue := mapItem.Value.(type) {
	case *Map:
		check := typedValue.Check()
		typeCheck.Violations = append(typeCheck.Violations, check.Violations...)
	case int:
		if _, ok := mapItem.Type.(int); !ok {
			violation := fmt.Sprintf(violationErrorMessage, mapItem.Key, mapItem.Position.AsCompactString(), typedValue, mapItem.Type)
			typeCheck.Violations = append(typeCheck.Violations, violation)
		}
		return typeCheck
	case string:
		if _, ok := mapItem.Type.(string); !ok {
			violation := fmt.Sprintf(violationErrorMessage, mapItem.Key, mapItem.Position.AsCompactString(), typedValue, mapItem.Type)
			typeCheck.Violations = append(typeCheck.Violations, violation)
		}
		return typeCheck
	}

	return typeCheck
}

func (d *DocumentSet) Check() TypeCheck { return TypeCheck{} }
func (d *Array) Check() TypeCheck       { return TypeCheck{} }
func (d *ArrayItem) Check() TypeCheck   { return TypeCheck{} }

func (as AnySchema) AssignType(_ Node) {}

func (s DocumentSchema) AssignType(node Node) {
	document, ok := node.(*Document)
	if !ok {
		panic("document schema assigntype called with non-document type")
	}
	document.Type = s.Allowed.Schema
	
	for _, val := range node.GetValues() {
		switch typedNode := val.(type) {
		case *Map:
			thisMapType, ok := s.Allowed.Schema.(*MapType)
			if !ok {
				typedNode.Type = &MapType{}
				// during typing we dont report error
				break
			}

			typedNode.Type = thisMapType

			AssignType(typedNode.Items)
			//schemaMapType.Schema.AssignType(typedNode)
		}
	}
}

func (m MapSchema) AssignType(node Node) {
	typedNode, ok := node.(*Map)
	if !ok {
		panic("map schema assigntype called with non-map type")
	}
	for _, mapItem := range typedNode.Items {
		for _, schemaItem := range m.Allowed.Items {
			if mapItem.Key == schemaItem.Key {
				// Found the matching schema section
				mapItem.Type = schemaItem.DefaultValue

				//desired call
				nodeValue, ok := mapItem.Value.(Node)
				if ok {
					schemaItem.Schema.AssignType(nodeValue)
				}


				// Check the children by calling AssignType recursively
				//mapItemAsNode := Node(mapItem)
				//oldAllowed := s.Allowed
				//s.Allowed = &DocumentType{&Document{Value: schemaItem.Value}}
				//s.AssignType(&mapItemAsNode)
				//s.Allowed = oldAllowed

				break
			}
		}

	}
}

func (m MapItemSchema) AssignType(node Node) {
	typedNode, ok := node.(*MapItem)
	if !ok {
		panic("mapItem schema assigntype called with non-mapItem type")
	}

	typedNode.Type = m.Allowed

	//Need to call AssignType on the children if its a map/array
}